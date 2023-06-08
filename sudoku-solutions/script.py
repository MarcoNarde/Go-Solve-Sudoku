import os

def convert_file(file_path):
    # Leggi il contenuto del file
    with open(file_path, 'r') as file:
        lines = file.readlines()

    # Crea una matrice vuota 9x9
    matrix = [[0] * 9 for _ in range(9)]

    # Popola la matrice con i valori corrispondenti
    for i in range(9):
        line = lines[i].split('|')
        line_values = line[0].strip().split()
        for j in range(9):
            matrix[i][j] = int(line_values[j])

    # Scrivi la matrice convertita nel file
    with open(file_path, 'w') as file:
        for i in range(9):
            for j in range(9):
                file.write(str(matrix[i][j]) + ' ')
            file.write('\n')

# Percorso della cartella contenente i file
folder_path = '.'

# Elabora tutti i file nella cartella
for filename in os.listdir(folder_path):
    file_path = os.path.join(folder_path, filename)
    if os.path.isfile(file_path):
        convert_file(file_path)
