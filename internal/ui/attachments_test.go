package ui

import (
	"errors"
	"os"
	"testing"

	"mailcat/internal/security"
)

// errWriter падает посреди записи (limit байт), при закрытии или и не пишет
// вовсе: так проверяется, что недописанный файл не остаётся на диске.
type errWriter struct {
	*os.File
	limit    int
	writeErr error
	closeErr error
}

func (w *errWriter) Write(p []byte) (int, error) {
	if w.limit >= 0 && len(p) > w.limit {
		n, _ := w.File.Write(p[:w.limit])
		return n, w.writeErr
	}
	return w.File.Write(p)
}

func (w *errWriter) Close() error {
	err := w.File.Close()
	if w.closeErr != nil {
		return w.closeErr
	}
	return err
}

// Чужой файл не удаляется, если путь в вызове разошёлся с писателем: ошибка
// записи не должна сносить файл, в который писал не этот writer.
func TestWriteAttachmentKeepsForeignFile(t *testing.T) {
	dir := t.TempDir()
	f, _, err := security.CreateUnique(dir, "own.bin")
	if err != nil {
		t.Fatal(err)
	}
	_, foreign, err := security.CreateUnique(dir, "foreign.bin")
	if err != nil {
		t.Fatal(err)
	}
	w := &errWriter{File: f, limit: 0, writeErr: errors.New("сбой")}
	if err := writeAttachment(foreign, w, []byte("0123456789")); err == nil {
		t.Fatal("writeAttachment() = nil, ожидалась ошибка")
	}
	if _, err := os.Stat(foreign); err != nil {
		t.Fatalf("чужой файл %s удалён: %v", foreign, err)
	}
}

func TestWriteAttachmentRemovesPartialFile(t *testing.T) {
	writeFail := errors.New("диск полон")
	closeFail := errors.New("сбой закрытия")
	for _, tc := range []struct {
		name      string
		limit     int
		writeErr  error
		closeErr  error
		wantErr   bool
		wantExist bool
	}{
		{name: "успех", limit: -1, wantExist: true},
		{name: "сбой записи", limit: 2, writeErr: writeFail, wantErr: true},
		{name: "сбой закрытия", limit: -1, closeErr: closeFail, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, path, err := security.CreateUnique(t.TempDir(), "att.bin")
			if err != nil {
				t.Fatal(err)
			}
			w := &errWriter{File: f, limit: tc.limit, writeErr: tc.writeErr, closeErr: tc.closeErr}
			err = writeAttachment(path, w, []byte("0123456789"))
			if (err != nil) != tc.wantErr {
				t.Fatalf("writeAttachment() = %v, want error %v", err, tc.wantErr)
			}
			_, statErr := os.Stat(path)
			if got := statErr == nil; got != tc.wantExist {
				t.Fatalf("файл существует = %v, want %v (stat: %v)", got, tc.wantExist, statErr)
			}
		})
	}
}
