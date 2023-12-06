package filedownloader

type File interface {
	Download(to string) error
}
