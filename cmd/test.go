package main

import (
	"log"
)

var global = 0

func doSomething(ceiling int) {
	s := make(chan struct{})
	defer close(s)

	global = ceiling

	for number := range generator(s) {
		log.Println(number)
		s <- struct{}{}
	}
}

func do(ceiling int) {
	global = ceiling

	numbers, signal := generatorWrapper()

	for number := range numbers {
		log.Println(number)

		signal()
	}
}

func generatorWrapper() (<-chan int, func()) {
	signal := make(chan struct{})
	//defer close(signal)

	return generator(signal), func() {
		signal <- struct{}{}
	}
}

func generator(signal chan struct{}) <-chan int {
	c := make(chan int)

	go func() {
		defer close(c)

		for i := 1; ; i += 1 {
			if i <= global {
				c <- i
			} else {
				break
			}
			<-signal
		}
	}()

	return c
}

func main() {
	do(10)

}
