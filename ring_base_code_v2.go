// Codigo exemplo para o trabaho de sistemas distribuidos (eleicao em anel)
// By Cesar De Rose - 2022

package main

import (
	"fmt"
	"sync"
)

type mensagem struct {
	tipo  int    // tipo da mensagem para fazer o controle do que fazer (eleicao, confirmacao da eleicao)
		// 2 - set receiving process as failed
		// 3 - set receiving process as functional
	corpo [3]int // conteudo da mensagem para colocar os ids (usar um tamanho ocmpativel com o numero de processos no anel)
}

var (
	chans = []chan mensagem{ // vetor de canais para formar o anel de eleicao - chan[0], chan[1] and chan[2] ...
		make(chan mensagem),
		make(chan mensagem),
		make(chan mensagem),
		make(chan mensagem),
	}
	controle = make(chan int)
	wg       sync.WaitGroup // wg is used to wait for the program to finish
)

func ElectionController(in chan int) {
	defer wg.Done()

	fmt.Println("CONTROLE: iniciando script de testes")

	var temp mensagem

	// 1. mudar o processo 0 - canal de entrada 3 - para falho (defini mensagem tipo 2 pra isto)
	temp.tipo = 2
	fmt.Printf("CONTROLE: mudar o processo 0 para falho\n")
	chans[3] <- temp
	fmt.Printf("CONTROLE: confirmacao %d\n", <-in) // receber e imprimir confirmacao

	// mudar o processo 1 - canal de entrada 0 - para falho (defini mensagem tipo 2 pra isto)
	temp.tipo = 2
	fmt.Printf("CONTROLE: mudar o processo 1 para falho\n")
	chans[0] <- temp
	fmt.Printf("CONTROLE: confirmacao %d\n", <-in) // receber e imprimir confirmacao

	// matar os outros processos
	fmt.Println("CONTROLE: encerrando todos os processos enviando mensagem de termino (codigo -10)")
	temp.tipo = -10
	for _, c := range chans {
		c <- temp
	}
}

func ElectionStage(TaskId int, in chan mensagem, out chan mensagem, leader int) {
	defer wg.Done()

	// variaveis locais que indicam se este processo e o lider e se esta ativo
	var actualLeader int
	var bFailed bool = false // todos inciam sem falha
	actualLeader = leader // indicacao do lider veio por paramatro

	var stop bool = false
	for !stop { // loop and serve until told to stop
		temp := <-in // ler mensagem
		fmt.Printf("%2d: recebi mensagem %d, [ %d, %d, %d ]\n", TaskId, temp.tipo, temp.corpo[0], temp.corpo[1], temp.corpo[2])
	
		// handle received message
		switch temp.tipo {
			case 2:
			{
				bFailed = true
				fmt.Printf("%2d: falho %v \n", TaskId, bFailed)
				fmt.Printf("%2d: lider atual %d\n", TaskId, actualLeader)
				controle <- -5
			}
			case 3:
			{
				bFailed = false
				fmt.Printf("%2d: falho %v \n", TaskId, bFailed)
				fmt.Printf("%2d: lider atual %d\n", TaskId, actualLeader)
				controle <- -5
			}
			case -10:
			{
				stop = true
			}
			default:
			{
				fmt.Printf("%2d: nao conheco este tipo de mensagem\n", TaskId)
				fmt.Printf("%2d: lider atual %d\n", TaskId, actualLeader)
			}
		}
	}

	fmt.Printf("%2d: terminei \n", TaskId)
}

func main() {
	wg.Add(5) // Add a count of four, one for each goroutine

	// criar os processos do anel de eleicao
	go ElectionStage(0, chans[3], chans[0], 0) // este e o lider
	go ElectionStage(1, chans[0], chans[1], 0) // nao e lider, e o processo 0
	go ElectionStage(2, chans[1], chans[2], 0) // nao e lider, e o processo 0
	go ElectionStage(3, chans[2], chans[3], 0) // nao e lider, e o processo 0
	fmt.Println("\nPRINCIPAL: Anel de processos criado")

	// criar o processo controlador
	go ElectionController(controle)
	fmt.Println("PRINCIPAL: Processo controlador criado")
	fmt.Println() // extra line break without warning

	wg.Wait() // Wait for the goroutines to finish\

	fmt.Println("\nPRINCIPAL: programa encerrado")
	fmt.Println() // extra line break without warning
}
