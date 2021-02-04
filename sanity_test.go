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

func Test01Plugin(t *testing.T) {
	// this test just tests that the plugin exists and terraform is not mad about
	// where is. Other places install it, we just want to make sure it'll work.

	tf := tftest.New(t)
	tf.Apply("testdata/plans/plugin-sanity-test.tf")
}
