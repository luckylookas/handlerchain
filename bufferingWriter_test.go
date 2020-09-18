package handlerchain

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

type MockResponseWriter struct {
}

func (w MockResponseWriter) Write(b []byte) (int, error) {
	return len(b), nil
}

func (w MockResponseWriter) Header() http.Header {
	return nil
}

func (w MockResponseWriter) WriteHeader(i int) {

}

func Test_bufferingHttpResponseWriter_Write(t *testing.T) {
	type fields struct {
		ResponseWriter http.ResponseWriter
		buffer         []byte
	}
	type args struct {
		b [][]byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []byte
		wantRet int
	}{
		{
			name: "should do simple write with empty buffer",
			fields: fields{
				ResponseWriter: MockResponseWriter{},
				buffer:         []byte{},
			},
			args: args{
				b: [][]byte{[]byte("abc")},
			},
			wantRet: 3,
			want:    []byte("abc"),
		},
		{
			name: "should do simple write with nil buffer",
			fields: fields{
				ResponseWriter: MockResponseWriter{},
				buffer:         nil,
			},
			args: args{
				b: [][]byte{[]byte("abc")},
			},
			wantRet: 3,
			want:    []byte("abc"),
		},
		{
			name: "should append multiple writes",
			fields: fields{
				ResponseWriter: MockResponseWriter{},
				buffer:         nil,
			},
			args: args{
				b: [][]byte{[]byte("abc"), []byte("def")},
			},
			wantRet: 3,
			want:    []byte("abcdef"),
		},
	}
	for _, tt := range tests {
		assertions := assert.New(t)
		t.Run(tt.name, func(t *testing.T) {
			w := bufferingHttpResponseWriter{
				ResponseWriter: tt.fields.ResponseWriter,
				buffer:         tt.fields.buffer,
			}

			for _, arg := range tt.args.b {
				got, err := w.Write(arg)
				assertions.NoError(err)
				assertions.Equal(tt.wantRet, got)
			}

			assertions.ElementsMatch(w.Buffer(), tt.want)
		})
	}
}
