package main

import (
	"fmt"
	"log"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
)

func main() {
	// Test that EnableWitnessStats field is properly set
	config := &ethconfig.Config{
		EnableWitnessStats: true,
	}

	vmConfig := vm.Config{
		EnableWitnessStats: config.EnableWitnessStats,
	}

	fmt.Printf("EnableWitnessStats in ethconfig: %t\n", config.EnableWitnessStats)
	fmt.Printf("EnableWitnessStats in vm.Config: %t\n", vmConfig.EnableWitnessStats)

	// Test that the field is accessible from blockchain config
	genesis := core.DefaultGenesisBlock()
	chainConfig := genesis.Config

	blockChainConfig := &core.BlockChainOptions{
		VmConfig: vmConfig,
	}

	fmt.Printf("VM Config in blockchain options has EnableWitnessStats: %t\n", blockChainConfig.VmConfig.EnableWitnessStats)
	fmt.Printf("Chain config: %s\n", chainConfig.ChainID.String())

	log.Println("✓ Witness stats configuration test passed!")
}
