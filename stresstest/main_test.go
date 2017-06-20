package main

import (
	"fmt"
	"testing"
)

func TestCalculateDepthFatal(t *testing.T) {
	value, err := calculateDepth(5, 3)
	if err != nil {
		t.Error(err)
	} else {
		fmt.Println("Found value:", value)
	}
}

func TestCalculateDepthSuccess(t *testing.T) {
	value, err := calculateDepth(100, 9)
	if err != nil {
		t.Error(err)
	} else {
		fmt.Println("Found value:", value)
	}
}
