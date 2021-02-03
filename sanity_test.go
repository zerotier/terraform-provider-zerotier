package main

import (
	"testing"

	"github.com/erikh/tftest"
)

func Test00Sanity(t *testing.T) {
	// this test principally exists to make sure terraform is working. If it is
	// not the first test; someone else added a test that runs before this and
	// should be scolded. :P

	tf := tftest.New(t)
	tf.Apply("testdata/plans/sanity-test.tf")
}
