//Todo: doc

package reader

type Reader interface {
	Read() ([][]byte,error)
}
