package internal

import (
	"os"
	"testing"
	"time"
)

func TestWorker(t *testing.T) {
	testDir := t.TempDir()

	params := Params{
		Interval: 1 * time.Second,
		QAUrls:   []string{"wss://tfchain.qa.grid.tf/ws"},
		TestUrls: []string{"wss://tfchain.test.grid.tf/ws"},
		MainUrls: []string{"wss://tfchain.grid.tf/ws"},
	}
	src := testDir + "/tf-autobuilder"
	dst := testDir + "/tf-zos"

	err := os.Mkdir(src, os.ModePerm)
	if err != nil {
		t.Error(err)
	}

	err = os.Mkdir(dst, os.ModePerm)
	if err != nil {
		t.Error(err)
	}

	worker, err := NewWorker(src, dst, params)
	if err != nil {
		t.Error(err)
	}

	t.Run("test_no_src_qa", func(t *testing.T) {
		err := worker.updateZosVersion("qa", worker.substrate["qa"])
		if err == nil {
			t.Errorf("update zos should fail")
		}
	})

	t.Run("test_no_src_test", func(t *testing.T) {
		_, err := os.Create(src + "/zos:v3.4.0-qa1.flist")
		if err != nil {
			t.Error(err)
		}

		err = worker.updateZosVersion("testing", worker.substrate["testing"])
		if err == nil {
			t.Errorf("update zos should fail for test, %v", err)
		}
	})

	t.Run("test_no_src_main", func(t *testing.T) {
		_, err = os.Create(src + "/zos:v3.1.1-rc2.flist")
		if err != nil {
			t.Error(err)
		}

		err = worker.updateZosVersion("production", worker.substrate["production"])
		if err == nil {
			t.Errorf("update zos should fail for main, %v", err)
		}
	})

	t.Run("test_params_wrong_url", func(t *testing.T) {
		params.QAUrls = []string{"wss://tfchain.qa1.grid.tf/ws"}

		worker, err = NewWorker(src, dst, params)
		if err != nil {
			t.Error(err)
		}
		err := worker.updateZosVersion("qa", worker.substrate["qa"])
		if err == nil {
			t.Errorf("update zos should fail")
		}
	})
}
