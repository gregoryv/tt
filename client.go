package tt

func NewClient() *Client {
	return &Client{}
}

type Client struct{
	// Public fields can all be modified before calling Run
	// Changing them afterwards should have no effect.

	// Server to connect to
	Server      *url.URL
}

func (c *Client) Run(ctx context.Context) error {
	return fmt.Errorf(": todo")
}
