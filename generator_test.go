package main

import (
	"os"
	"testing"
)

func TestTemplateGenerator_run(t *testing.T) {

	os.RemoveAll("repos/")
	// process all the template
	reports := Generartor().run()

	for _, r := range reports {
		if r.err != nil {
			t.Errorf("the report must not contains failures: %v", r.err)
		}
	}
}
