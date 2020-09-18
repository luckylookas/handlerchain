package handlerchain

type errstring string
func (e errstring) Error() string {
	return string(e)
}
