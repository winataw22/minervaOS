package backend

// import (
// 	"io/ioutil"
// 	"os"
// 	"path/filepath"

// 	. "github.com/onsi/ginkgo"
// 	. "github.com/onsi/gomega"
// )

// var _ = Describe("Lock Operations", func() {
// 	It("locks a file path", func() {
// 		dir, err := ioutil.TempDir("", "")
// 		Expect(err).ToNot(HaveOccurred())
// 		defer os.RemoveAll(dir)

// 		// create a dummy file to lock
// 		path := filepath.Join(dir, "x")
// 		f, err := os.OpenFile(path, os.O_RDONLY|os.O_CREATE, 0666)
// 		Expect(err).ToNot(HaveOccurred())
// 		err = f.Close()
// 		Expect(err).ToNot(HaveOccurred())

// 		// now use it to lock
// 		m, err := NewFileLock(path)
// 		Expect(err).ToNot(HaveOccurred())

// 		err = m.Lock()
// 		Expect(err).ToNot(HaveOccurred())
// 		err = m.Unlock()
// 		Expect(err).ToNot(HaveOccurred())
// 	})

// 	It("locks a folder path", func() {
// 		dir, err := ioutil.TempDir("", "")
// 		Expect(err).ToNot(HaveOccurred())
// 		defer os.RemoveAll(dir)

// 		// use the folder to lock
// 		m, err := NewFileLock(dir)
// 		Expect(err).ToNot(HaveOccurred())

// 		err = m.Lock()
// 		Expect(err).ToNot(HaveOccurred())
// 		err = m.Unlock()
// 		Expect(err).ToNot(HaveOccurred())
// 	})
// })
