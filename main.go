package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"
)

type Solution struct { //Square
	x int // x-coordinate of the cell
	y int // y-coordinate of the cell
	v int // content of the cell in (x,y)
}

type Coordinates struct {
	Row, Col int
}

type TemporalElim struct {
	eliminatedValues [9]int
	x                int
	y                int
}

type Solver struct {
	notContain        [9][9]chan int
	done              chan Solution
	tempElim          chan TemporalElim
	tempBoard         [9][9][9]int
	solvedSudokuBoard [9][9]int
	callGrid          [9]chan int
	callRow           [9]chan int
	callCol           [9]chan int
	callLockedGrid    [9]chan int
}

func SolveSquare(x, y int, notContain <-chan int, done chan Solution, tempElim chan<- TemporalElim) {
	var eliminated [9]bool

	for n := range notContain {
		eliminated[n-1] = true
		var c, s int
		var elimValues [9]int

		for i, v := range eliminated {
			if v {
				c++
				elimValues[i] = 0
			} else {
				s = i + 1
				elimValues[i] = i + 1
			}
		}

		if c == 8 {
			done <- Solution{x, y, s}
			for range notContain {
			}
		} else {
			tempElim <- TemporalElim{elimValues, x, y}
		}
	}
}

func NewSolver() (S *Solver) {
	S = &Solver{done: make(chan Solution), tempElim: make(chan TemporalElim)}
	for y := 0; y < 9; y++ {
		for x := 0; x < 9; x++ {
			S.notContain[y][x] = make(chan int)
			go SolveSquare(x, y, S.notContain[y][x], S.done, S.tempElim)
		}
	}
	S.StartGridCheck()
	S.StartRowCheck()
	S.StartColCheck()
	S.StartLockedCandCheck()
	return S
}

func (S *Solver) SolveSudoku(enableLockedCand *int) [9][9]int {
	responses := 0
	countLockedGrid := 0
	countIteration := 0
	for {
		select {
		case u := <-S.done:
			go S.Eliminate(u)
			S.solvedSudokuBoard[u.y][u.x] = u.v

			var array [9]int

			for i := 0; i < len(array); i++ {
				if i == u.v-1 {
					array[i] = u.v
				} else {
					array[i] = 0
				}
			}

			S.tempBoard[u.y][u.x] = array

			fmt.Printf("Step: %d %#v\n", responses, u)
			responses++

			if responses == 81 {
				return S.solvedSudokuBoard
			}

		case elim := <-S.tempElim:
			S.tempBoard[elim.y][elim.x] = elim.eliminatedValues
		case <-time.After(1 * time.Second):
			countIteration++
			if countIteration > 50 {
				return S.solvedSudokuBoard
			}

			// Inizializza il generatore di numeri casuali con un seed diverso ad ogni esecuzione
			rand.Seed(time.Now().UnixNano())

			// Genera un numero casuale tra 0 e 8 compresi
			randomChoice := rand.Intn(9)

			const gridCount = 9

			fmt.Println("Grids Check")
			for i := 0; i < gridCount; i++ {
				S.callGrid[i] <- 1
			}
			for i := 0; i < gridCount; i++ {
				S.callRow[i] <- 1
			}
			for i := 0; i < gridCount; i++ {
				S.callCol[i] <- 1
			}

			if *enableLockedCand == 0 || countLockedGrid < 5 {
				countLockedGrid++
			} else {
				countLockedGrid = 0
				fmt.Println("Locked Check")
				S.callLockedGrid[randomChoice] <- 1
				S.callLockedGrid[(randomChoice+1)%gridCount] <- 1
				S.callLockedGrid[(randomChoice+2)%gridCount] <- 1
				S.callLockedGrid[(randomChoice+3)%gridCount] <- 1
			}

		}
	}

	return S.solvedSudokuBoard
}

func (S *Solver) Eliminate(u Solution) {
	// row
	for x := 0; x < 9; x++ {
		if x != u.x {
			S.notContain[u.y][x] <- u.v
		}
	}
	// column
	for y := 0; y < 9; y++ {
		if y != u.y {
			S.notContain[y][u.x] <- u.v
		}
	}
	// 3x3 group
	sX, sY := u.x/3*3, u.y/3*3 // group start coordinates
	for y := sY; y < sY+3; y++ {
		for x := sX; x < sX+3; x++ {
			if x != u.x || y != u.y {
				S.notContain[y][x] <- u.v
			}
		}
	}
}

func (S *Solver) Set(x, y, v int) {
	go func() {
		for i := 1; i <= 9; i++ {
			if i != v {
				S.notContain[y][x] <- i
			}
		}
	}()
}

func (S *Solver) CheckGrid(i int, j int, activate <-chan int) {
	// Calcolo degli indici di inizio per la griglia corrente
	startRow := i * 3 // es i=1 ->  sr = 3
	startCol := j * 3

	for range activate {

		var callWithOneValue [3][3]int

		// Controllo le celle che contengono almeno 2 valori
		for row := startRow; row < startRow+3; row++ {
			for col := startCol; col < startCol+3; col++ {
				var count = 0
				for i := 0; i < 9; i++ {
					if S.tempBoard[row][col][i] != 0 {
						count++
					}
				}
				if count >= 2 {
					callWithOneValue[row-startRow][col-startCol] = 1
				}
			}
		}

		// Mappa per conteggiare la frequenza dei valori
		valueCount := make(map[int]int)
		cellTrack := make(map[int][]Coordinates)

		// Scorrimento delle caselle nella griglia corrente e conteggio dei valori
		for row := startRow; row < startRow+3; row++ {
			for col := startCol; col < startCol+3; col++ {
				if callWithOneValue[row-startRow][col-startCol] == 1 {
					for i := 0; i < 9; i++ {
						if S.tempBoard[row][col][i] != 0 {
							valueCount[i]++
							cellTrack[i] = append(cellTrack[i], Coordinates{row, col})
						}
					}
				}
			}
		}

		// Verifica se esiste un unico valore che può essere assegnato
		for value, c := range valueCount {
			if c == 1 {
				var cell = cellTrack[value]
				if len(cell) == 1 {
					for i := 1; i <= 9; i++ {
						if i != (value + 1) {
							S.notContain[cell[0].Row][cell[0].Col] <- i
						}
					}
				}
			}
		}
	}
}

func (S *Solver) StartGridCheck() {
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			S.callGrid[(i*3)+j] = make(chan int)
			go S.CheckGrid(i, j, S.callGrid[(i*3)+j])
		}
	}
}

func (S *Solver) StartLockedCandCheck() {
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			S.callLockedGrid[(i*3)+j] = make(chan int)
			go S.CheckLockedCandidate(i, j, S.callLockedGrid[(i*3)+j])
		}
	}
}

func (S *Solver) StartRowCheck() {
	for i := 0; i < 9; i++ {
		S.callRow[i] = make(chan int)
		go S.CheckRow(i, S.callRow[i])
	}
}

func (S *Solver) StartColCheck() {
	for i := 0; i < 9; i++ {
		S.callCol[i] = make(chan int)
		go S.CheckCol(i, S.callCol[i])
	}
}

func (S *Solver) CheckRow(rowN int, activate <-chan int) {
	for range activate {

		var cellWithOneValue [9]int

		// Controllo le celle che contengono almeno 2 valori
		for col := 0; col < 9; col++ {
			var count = 0
			for i := 0; i < 9; i++ {
				if S.tempBoard[rowN][col][i] != 0 {
					count++
				}
			}
			if count >= 2 {
				cellWithOneValue[col] = 1
			}
		}

		// Mappa per conteggiare la frequenza dei valori
		valueCount := make(map[int]int)
		cellTrack := make(map[int][]int)

		// Scorrimento delle caselle nella griglia corrente e conteggio dei valori
		for col := 0; col < 9; col++ {
			if cellWithOneValue[col] == 1 {
				for i := 0; i < 9; i++ {
					if S.tempBoard[rowN][col][i] != 0 {
						valueCount[i]++
						cellTrack[i] = append(cellTrack[i], col)
					}
				}
			}
		}

		// Verifica se esiste un unico valore che può essere assegnato
		for value, c := range valueCount {
			if c == 1 {
				var cell = cellTrack[value]
				if len(cell) == 1 {
					for v := 1; v <= 9; v++ {
						if v != (value + 1) {
							S.notContain[rowN][cell[0]] <- v
						}
					}
				}
			}
		}
	}
}

func (S *Solver) CheckLockedCandidate(i int, j int, activate <-chan int) {
	// Calcolo degli indici di inizio per la griglia corrente
	startRow := i * 3 // es i=1 ->  sr = 3
	startCol := j * 3

	for range activate {

		var callWithOneValue [3][3]int

		// Controllo le celle che contengono almeno 2 valori
		for row := startRow; row < startRow+3; row++ {
			for col := startCol; col < startCol+3; col++ {
				var count = 0
				for i := 0; i < 9; i++ {
					if S.tempBoard[row][col][i] != 0 {
						count++
					}
				}
				if count >= 2 {
					callWithOneValue[row-startRow][col-startCol] = 1
				}
			}
		}

		// Implementazione Locked candidate
		valueCount2 := make(map[int]int)
		cellTrack2 := make(map[int][]Coordinates)

		// Scorrimento delle caselle nella griglia corrente e conteggio dei valori
		for row := startRow; row < startRow+3; row++ {
			for col := startCol; col < startCol+3; col++ {
				if callWithOneValue[row-startRow][col-startCol] == 1 {
					for i := 0; i < 9; i++ {
						if S.tempBoard[row][col][i] != 0 {
							valueCount2[i]++
							cellTrack2[i] = append(cellTrack2[i], Coordinates{row, col})
						}
					}
				}
			}
		}

		for value, c := range valueCount2 {
			if c == 3 || c == 2 {
				var cells = cellTrack2[value]
				// Controllo che ogni cella sia nella stessa riga o colonna
				var row, col int
				var i = 0
				rowColCount := make(map[int]int)
				for _, cell := range cells {
					if i == 0 {
						row = cell.Row
						col = cell.Col
						rowColCount[row]++
						rowColCount[col]++
					} else {
						if cell.Row == row {
							rowColCount[row]++
						}
						if cell.Col == col {
							rowColCount[col]++
						}
					}
					i++
				}
				if rowColCount[row] == c {
					// Invio value a tutta la riga (tranne alle celle di questa griglia)
					for c := 0; c < 9; c++ {
						if c < startCol || c > startCol+2 {
							S.notContain[row][c] <- value + 1
						}
					}
				} else if rowColCount[col] == c {
					// Invio value a tutta la riga (tranne alle celle di questa griglia)
					for c := 0; c < 9; c++ {
						if c < startRow || c > startRow+2 {
							S.notContain[c][col] <- value + 1
						}
					}
				}
			}
		}
	}
}

func (S *Solver) CheckCol(colN int, activate chan int) {
	for range activate {

		var cellWithOneValue [9]int

		// Controllo le celle che contengono almeno 2 valori
		for row := 0; row < 9; row++ {
			var count = 0
			for i := 0; i < 9; i++ {
				if S.tempBoard[row][colN][i] != 0 {
					count++
				}
			}
			if count >= 2 {
				cellWithOneValue[row] = 1
			}
		}

		// Mappa per conteggiare la frequenza dei valori
		valueCount := make(map[int]int)
		cellTrack := make(map[int][]int)

		// Scorrimento delle caselle nella griglia corrente e conteggio dei valori
		for row := 0; row < 9; row++ {
			if cellWithOneValue[row] == 1 {
				for i := 0; i < 9; i++ {
					if S.tempBoard[row][colN][i] != 0 {
						valueCount[i]++
						cellTrack[i] = append(cellTrack[i], row)
					}
				}
			}
		}

		// Verifica se esiste un unico valore che può essere assegnato
		for value, c := range valueCount {
			if c == 1 {
				var cell = cellTrack[value]
				if len(cell) == 1 {
					for v := 1; v <= 9; v++ {
						if v != (value + 1) {
							S.notContain[cell[0]][colN] <- v
						}
					}
				}
			}
		}
	}
}

func PrettyPrintSudoku(sudoku [9][9]int) {
	for row := 0; row < 9; row++ {
		if row%3 == 0 && row != 0 {
			fmt.Println("---------------------")
		}
		for col := 0; col < 9; col++ {
			if col%3 == 0 && col != 0 {
				fmt.Print("| ")
			}
			fmt.Printf("%d ", sudoku[row][col])
		}
		fmt.Println()
	}
}

// ReadSudokuFromFile legge il file specificato e restituisce la matrice sudoku
func ReadSudokuFromFile(filePath string) ([9][9]int, error) {
	var sudoku [9][9]int

	file, err := os.Open(filePath)
	if err != nil {
		return sudoku, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	row := 0
	for scanner.Scan() {
		line := scanner.Text()
		values := strings.Split(line, " ")

		for col, val := range values {
			num, err := strconv.Atoi(val)
			if err != nil {
				return sudoku, err
			}
			sudoku[row][col] = num
		}

		row++
	}

	if err := scanner.Err(); err != nil {
		return sudoku, err
	}

	return sudoku, nil
}

func CompareSudokuMatrices(matrix1, matrix2 [9][9]int) []Coordinates {
	var differences []Coordinates
	for row := 0; row < 9; row++ {
		for col := 0; col < 9; col++ {
			if matrix1[row][col] != matrix2[row][col] {
				position := Coordinates{Row: row, Col: col}
				differences = append(differences, position)
			}
		}
	}

	return differences
}

func main() {

	// Definisci un flag di tipo string per il percorso del file
	filePath := flag.String("file", "", "Path to the input file")
	solutionFilePath := flag.String("solution", "", "Path to the solution file")
	enableLockCand := flag.Int("enableLockCand", 0, "Enable to use the locked candidate technique (0 or 1)")

	showHelp := flag.Bool("help", false, "Display the list of available flags")
	flag.Parse()

	// Check if the help flag is set
	if *showHelp {
		flag.Usage() // Display the list of available flags
		return
	}

	if *filePath == "" {
		flag.Usage()
		log.Fatal("Input file path is missing")
	}

	sudoku, err := ReadSudokuFromFile(*filePath)
	if err != nil {
		log.Fatalf("Error reading file: %v", err)
	}

	startTime := time.Now()
	solver := NewSolver()
	setInitialValues(solver, sudoku)
	solution := solver.SolveSudoku(enableLockCand)
	elapsedTime := time.Since(startTime).Seconds()

	printSolutionAndTime(solution, elapsedTime)

	if *solutionFilePath != "" {
		compareWithSolutionFile(solution, *solutionFilePath)
	}
}

func setInitialValues(solver *Solver, sudoku [9][9]int) {
	for y := 0; y < 9; y++ {
		for x := 0; x < 9; x++ {
			if sudoku[y][x] != 0 {
				solver.Set(x, y, sudoku[y][x])
			}
		}
	}
}

func printSolutionAndTime(solution [9][9]int, elapsedTime float64) {
	fmt.Println("Last solution")
	PrettyPrintSudoku(solution)
	fmt.Printf("Elapsed time: %.4f s\n", elapsedTime)
}

func compareWithSolutionFile(solution [9][9]int, solutionFilePath string) {
	fmt.Println("Comparing result with the actual solution...")
	realSolution, err := ReadSudokuFromFile(solutionFilePath)
	if err != nil {
		log.Fatalf("Error reading the solution file: %v", err)
	}

	diffCells := CompareSudokuMatrices(solution, realSolution)
	percent := float64(81-len(diffCells)) / 81.0 * 100.0

	if len(diffCells) == 0 {
		fmt.Println("The solution is 100% correct!")
	} else {
		fmt.Printf("The solution has %d different cells, so it's %.2f%% correct\n", len(diffCells), percent)
		fmt.Println("Coordinates of differing cells:")
		for _, coord := range diffCells {
			fmt.Printf("Cell at position (%d, %d)\n", coord.Row, coord.Col)
		}
	}
}
