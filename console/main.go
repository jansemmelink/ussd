package main

import "bitbucket.org/vservices/ms-vservices-ussd/examples/pcm"

func main() {
	pcmInit := pcm.Item()
	Run(pcmInit)
}
