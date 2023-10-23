// Codigo exemplo para o trabaho de sistemas distribuidos (eleicao em anel)
// By Cesar De Rose - 2022

package main

import (
	"fmt"
	"sync"
)

type mensagem struct {
	tipo  int    // tipo da mensagem para fazer o controle do que fazer (eleicao, confirmacao da eleicao etc.)
	corpo [4]int // conteudo da mensagem para colocar os ids (usar um tamanho ocmpativel com o numero de processos no anel)
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

	var temp mensagem

	// 1. mudar o processo 3 - canal de entrada 2 - para falho (defini mensagem tipo 2 pra isto)
	fmt.Printf("CONTROLADOR: mudar o processo 3 para falho\n")
	temp.tipo = 2
	chans[2] <- temp
	var result int = <- in
	fmt.Printf("CONTROLADOR: sucesso = %v\n", result==2) // receber e imprimir confirmacao

	fmt.Println("--------------------------------------------------------------")

	// 2. solicitar ao processo 1 - canal de entrada 0 - para iniciar uma eleicao pois detectou o processo 3 como falho (deve vencer o processo 2)
	fmt.Printf("CONTROLADOR: solicitar ao processo 1 para iniciar eleicao pois detectou o processo 3 como falho\n")
	temp.tipo = 4
	temp.corpo = [4]int{3, 3, 3, 3}
	chans[0] <- temp
	result = <- in
	fmt.Printf("CONTROLADOR: sucesso = %v\n", result==4) // receber e imprimir confirmacao

	fmt.Println("--------------------------------------------------------------")

	// 3. reativar o processo 3 - canal de entrada 2. ele deve convocar uma eleicao e vencer
	fmt.Printf("CONTROLADOR: reativar o processo 3\n")
	temp.tipo = 3
	chans[2] <- temp
	result = <- in
	fmt.Printf("CONTROLADOR: sucesso = %v\n", result==3) // receber e imprimir confirmacao
	fmt.Println("--------------------------------------------------------------")

	// 4. encerrar os outros processos para terminar o programa
	fmt.Println("CONTROLADOR: encerrando todos os processos enviando mensagem de termino (codigo 10)")
	temp.tipo = 10
	for i, c := range chans {
		c <- temp
		result = <- in
		fmt.Printf("CONTROLADOR: confirmacao de termino do processo %d = %v\n", i, result==10)
	}

	fmt.Println("CONTROLADOR: processo controlador concluido")
	fmt.Println() // extra line break without warning
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
		fmt.Printf("\t%2d: recebi mensagem %d, [ %d, %d, %d, %d ]\n", TaskId, temp.tipo, temp.corpo[0], temp.corpo[1], temp.corpo[2], temp.corpo[3])
	
		// handle received message
		switch temp.tipo {
			case 0:  // VOTE REQUEST
			{
				if !bFailed {                   // if this process is failed, ignore and just don't put its ID in the message (don't vote)
					temp.corpo[TaskId] = TaskId // put id in the message body
					fmt.Printf("\t%2d: votei\n", TaskId)
				} else { fmt.Printf("\t%2d: nao votei, pois estou inativo\n", TaskId) }
				fmt.Printf("\t%2d: lider atual = %d\n", TaskId, actualLeader)
				out <- temp                     // pass message on to the next node
			}
			case 1:  // ELECTION WINNER CONFIRMATION
			{
				if !bFailed {                    // if this process is failed, ignore and just don't update its current leader
					actualLeader = temp.corpo[0] // update current leader
					fmt.Printf("\t%2d: atualizei meu lider para %d\n", TaskId, actualLeader)
				} else { fmt.Printf("\t%2d: nao atualizei meu lider, pois estou inativo\n", TaskId) }
				fmt.Printf("\t%2d: lider atual = %d\n", TaskId, actualLeader)
				out <- temp                      // pass message on to the next node
			}
			case 2:  // SET FAILURE (COMMAND RECEIVED FROM THE CONTROLLER PROCESS)
			{
				bFailed = true
				fmt.Printf("\t%2d: falho = %v \n", TaskId, bFailed)
				fmt.Printf("\t%2d: lider atual = %d\n", TaskId, actualLeader)
				controle <- 2
			}
			case 3:  // UNSET FAILURE (COMMAND RECEIVED FROM THE CONTROLLER PROCESS)
			{
		
				bFailed = false
				fmt.Printf("\t%2d: falho %v \n", TaskId, bFailed)
				fmt.Printf("\t%2d: lider atual = %d\n", TaskId, actualLeader)
				fmt.Printf("\t%2d: iniciando eleicao por volta a falha\n", TaskId)
				performElection(TaskId, in, out, &actualLeader)
				controle <- 3
					
			}
			case 4:  // INITIATE ELECTION (COMMAND RECEIVED FROM THE CONTROLLER PROCESS)
			{
				fmt.Printf("\t%2d: detectei falha no nodo %d\n", TaskId, temp.corpo[0])
				performElection(TaskId, in, out, &actualLeader)
				controle <- 4 // confirm that the election has been concluded to the controller
			}
			case 10: // TERMINATION REQUEST (COMMAND RECEIVED FROM THE CONTROLLER PROCESS)
			{ 
				stop = true 
				controle <- 10
			}
			default: // UNKNOWN COMMAND
			{
				fmt.Printf("\t%2d: nao conheco este tipo de mensagem\n", TaskId)
				fmt.Printf("\t%2d: lider atual = %d\n", TaskId, actualLeader)
			}
		}
	}

	fmt.Printf("\t%2d: terminei \n", TaskId)
}

func performElection(TaskId int, in chan mensagem, out chan mensagem, actualLeader *int) { 
	//TODO: missing verification if im the one who faild
	electionMsg := mensagem {
		tipo: 0, // 0 = election convocation
		corpo: [4]int{-1, -1, -1, -1}, // "empty" body (garbage)
	}
	electionMsg.corpo[TaskId] = TaskId // add my vote
	out <- electionMsg // send in the ring
	fmt.Printf("\t%2d: adicionei meu voto e enviei mensagem de eleicao no anel. Aguardando resultados...\n\n", TaskId)

	// wait for results to come back, calculate winner, update my leader and construct & send confirmation message in the ring
	result := <-in // wait for results
	if result.tipo != 0 {
		fmt.Printf("\t%2d: recebi mensagem inesperada como resultado da eleicao (esperava codigo 0, mas recebi %d)\n", TaskId, result.tipo)
		controle <- -4
		return
	}
	fmt.Printf("\n\t%2d: recebi os votos dos demais processos: [ %d, %d, %d, %d ]\n", TaskId, result.corpo[0], result.corpo[1], result.corpo[2], result.corpo[3])
	winner := highestValue(result.corpo[:])
	confirmationMsg := mensagem {
		tipo: 1,
		corpo: [4]int{winner, winner, winner, winner},
	}
	fmt.Printf("\t%2d: calculei o vencedor como o nodo %d\n", TaskId, winner)
	*actualLeader = winner // update my leader
	fmt.Printf("\t%2d: atualizei meu lider para %d\n", TaskId, *actualLeader)
	fmt.Printf("\t%2d: lider atual = %d\n", TaskId, *actualLeader)
	fmt.Printf("\t%2d: enviei mensagem de confirmacao de novo lider no anel. Aguardando resultados...\n\n", TaskId)
	out <- confirmationMsg

	// wait for confirmation of leaders updated to arrive
	confirmationResult := <- in
	if confirmationResult.tipo != 1 {
		fmt.Printf("\t%2d: recebi mensagem inesperada como confirmacao de atualizacao de lider (esperava codigo 1, mas recebi %d)\n", TaskId, confirmationResult.tipo)
		controle <- -4
		return
	}
	fmt.Printf("\n\t%2d: recebi confirmacao de que todos lideres foram atualizados e a eleicao foi concluida com sucesso!\n", TaskId)
}

func highestValue(values[]int) int {
	highest := values[0]
	for i:=1; i<len(values); i++ {
		if values[i] > highest {
			highest = values[i]
		}
	}
	return highest
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
}
