package services

import (
	"context"
	"github.com/machinebox/graphql"
)

type ExpoUserAccount struct {
	Id       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

func FetchExpoUserAccountInformations(expoToken string) (*ExpoUserAccount, error) {
	client := graphql.NewClient("https://api.expo.dev/graphql")

	req := graphql.NewRequest(`
		query GetCurrentUserAccount {
			me {
				id
				username
				email
			}
		}
	`)
	req.Header.Set("Authorization", "Bearer "+expoToken)

	ctx := context.Background()
	var resp struct {
		Me ExpoUserAccount `json:"me"`
	}
	if err := client.Run(ctx, req, &resp); err != nil {
		return nil, err
	}

	return &resp.Me, nil
}
