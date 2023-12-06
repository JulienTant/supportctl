package git

import (
	"context"

	gitlib "github.com/go-git/go-git/v5"
)

type LibClient struct{}

func NewLibClient() *LibClient {
	return &LibClient{}
}

func (c *LibClient) Clone(ctx context.Context, repo, dest string) error {
	_, err := gitlib.PlainCloneContext(ctx, dest, false, &gitlib.CloneOptions{
		URL: repo,
	})
	return err
}
