package main

import (
	"fmt"
	"math/rand"
	"time"
)

type Solution struct { //Square
	x int // x-coordinate of the cell
	y int // y-coordinate of the cell
	v int // content of the cell in (x,y)
}

type Coordinates struct {
	X, Y int
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
			fmt.Printf("X: %d, Y: %d, SOLUTION: %d\n", x, y, s)
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
	return S
}

func (S *Solver) SolveSudoku() [9][9]int {
	responses := 0
	//startGridCheck := false
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
			fmt.Println(S.solvedSudokuBoard)
			responses++

			if responses == 81 {
				return S.solvedSudokuBoard
			}

		case elim := <-S.tempElim:
			S.tempBoard[elim.y][elim.x] = elim.eliminatedValues
		case <-time.After(2 * time.Second):
			/*if !startGridCheck {
				for i := 0; i < 9; i++ {
					for j := 0; j < 9; j++ {
						fmt.Printf("cella [%d][%d] = %v\n", i, j, S.tempBoard[i][j])
					}
				}
				startGridCheck = true
				S.StartGridCheck()
			}*/
			// Inizializza il generatore di numeri casuali con un seed diverso ad ogni esecuzione
			rand.Seed(time.Now().UnixNano())

			// Genera un numero casuale tra 0 e 8 compresi
			randomNumber := rand.Intn(9)
			S.callGrid[randomNumber] <- 1
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

	for {
		<-activate

		fmt.Printf("Start grid check for [%d,%d]\n\n", i, j)

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
					//fmt.Printf("Cella [%d,%d] inclusa", row, col)
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

		/*for key, value := range valueCount {
			fmt.Printf("Chiave: %d, Valore: %d\n da griglia [%d,%d]", key, value, i, j)
		}
		for key, coordinates := range cellTrack {
			fmt.Printf("Chiave: %d\n", key)
			for _, coord := range coordinates {
				fmt.Printf("Coordinate: X=%d, Y=%d\n", coord.X, coord.Y)
			}
		}*/

		// Verifica se esiste un unico valore che può essere assegnato
		for value, c := range valueCount {
			//fmt.Printf("Value: %d, C: %d", value, c)
			if c == 1 {
				var cell = cellTrack[value]
				//fmt.Printf("Lenght: %d", len(cell))
				if len(cell) == 1 {
					for i := 1; i <= 9; i++ {
						if i != (value + 1) {
							S.notContain[cell[0].X][cell[0].Y] <- i
						}
					}
					//fmt.Printf("Unique value %d can be assigned to cell [%d,%d] in grid (%d,%d)\n", value+1, cell[0].X, cell[0].Y, i, j)
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

// Works for easy but not medium
func main() {

	var sudoku = [9][9]int{
		{2, 0, 0, 0, 0, 0, 6, 9, 0},
		{0, 5, 0, 0, 0, 3, 0, 0, 0},
		{1, 7, 0, 0, 0, 9, 4, 0, 5},
		{0, 0, 3, 0, 2, 5, 0, 1, 8},
		{0, 0, 0, 0, 4, 0, 0, 0, 0},
		{7, 2, 0, 3, 8, 0, 5, 0, 0},
		{5, 0, 2, 6, 0, 0, 0, 4, 1},
		{0, 0, 0, 5, 0, 0, 0, 7, 0},
		{0, 6, 7, 0, 0, 0, 0, 0, 3},
	}

	startTime := time.Now()
	solver := NewSolver()
	for y := 0; y < 9; y++ {
		for x := 0; x < 9; x++ {
			if sudoku[y][x] != 0 {
				solver.Set(x, y, sudoku[y][x])
			}
		}
	}
	var solution = solver.SolveSudoku()
	elapsedTime := time.Since(startTime)
	fmt.Println("Last solution")
	fmt.Println(solution)
	fmt.Println("Elapsed time:", elapsedTime)
}
