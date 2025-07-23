# Emulador MalcoNES

Um emulador do Nintendo Entertainment System (NES) escrito em [Go](https://golang.org/) utilizando a biblioteca [Ebiten](https://ebitengine.org/).

## Sobre

Este projeto implementa um emulador funcional do NES utilizando a linguagem Go.
Ele implementa os Mappers iNes 0, 1, 2 e 4

## Requisitos

- Go 1.23 ou superior
- Depêndencias de sistema utilizados pelo [Ebiten](https://ebitengine.org/en/documents/install.html)

## Execução

Clone o repositório em um local conveniente em seu sistema, entre na pasta clonada e execute os seguintes comandos:

```bash
go mod tidy
go run .
```

Por enquanto o camanho da rom .nes está hardcoded no arquivo main.go enquanto fazemos uma interface mais amigável.

## Controles
<pre>
W | A | S | D      ->    Controles Direcionais <br>
J                  ->             A            <br>
K                  ->             B            <br>
Espaço             ->           Start          <br>
Z                  ->           Select         <br>
</pre>
