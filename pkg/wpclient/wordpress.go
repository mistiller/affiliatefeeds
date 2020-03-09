package wpclient

import (
	"fmt"
	"log"
	"net/http"

	"github.com/sogko/go-wordpress"
)

// WPClient allows to access wordpress and manipulate objects underlying WooCommerce like media and posts
type WPClient struct {
	baseURL  string
	user     string
	password string
	client   *wordpress.Client
}

// New returns pointer to new WPClient
func New(host, apiVersion, user, password string) (c *WPClient, err error) {
	c = new(WPClient)
	url := fmt.Sprintf("%s/wp-json/wp/%s", host, apiVersion)
	log.Println(url)
	c.client = wordpress.NewClient(&wordpress.Options{
		BaseAPIURL: url,
		Username:   user,
		Password:   password,
	})

	_, resp, b, _ := c.client.Users().Me(nil)
	if resp.StatusCode != http.StatusOK {
		return c, fmt.Errorf("Failed to create Wordpress client - %s", string(b))
	}

	return c, nil
}

func (wp *WPClient) GetMedia() error {
	media, resp, body, err := wp.client.Media().List(nil)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Failed to list Wordpress media - %s -%s", resp.Status, string(body))
	}
	fmt.Println(media[0])

	return nil
}
