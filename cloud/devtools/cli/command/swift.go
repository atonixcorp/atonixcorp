package command

import (
	"context"
	"fmt"
	"os"

	sdk "atonixcorp/cloud/devtools/sdk/go"
)

func CreateContainer(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: atcloud create container <name>")
		return
	}

	client := sdk.NewClient(sdk.ClientOptions{
		Endpoint: os.Getenv("ATCLOUD_ENDPOINT"),
		Token:    os.Getenv("ATCLOUD_TOKEN"),
	})

	res, err := client.CreateContainer(context.Background(), args[0])
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("Container created:", res.Name)
}

func DeleteContainer(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: atcloud delete container <name>")
		return
	}

	client := sdk.NewClient(sdk.ClientOptions{
		Endpoint: os.Getenv("ATCLOUD_ENDPOINT"),
		Token:    os.Getenv("ATCLOUD_TOKEN"),
	})

	err := client.DeleteContainer(context.Background(), args[0])
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("Container deleted:", args[0])
}

func ListContainers() {
	client := sdk.NewClient(sdk.ClientOptions{
		Endpoint: os.Getenv("ATCLOUD_ENDPOINT"),
		Token:    os.Getenv("ATCLOUD_TOKEN"),
	})
	containers, err := client.ListContainers(context.Background())
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	for _, container := range containers {
		fmt.Println(container.Name)
	}
}

func UploadObject(args []string) {
	if len(args) < 3 {
		fmt.Println("Usage: atcloud upload object <container> <name> <data>")
		return
	}

	container := args[0]
	name := args[1]
	data := []byte(args[2])

	client := sdk.NewClient(sdk.ClientOptions{
		Endpoint: os.Getenv("ATCLOUD_ENDPOINT"),
		Token:    os.Getenv("ATCLOUD_TOKEN"),
	})

	res, err := client.UploadObject(context.Background(), container, name, data)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("Object uploaded:", res.Name)
}

func DeleteObject(args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: atcloud delete object <container> <name>")
		return
	}

	container := args[0]
	name := args[1]

	client := sdk.NewClient(sdk.ClientOptions{
		Endpoint: os.Getenv("ATCLOUD_ENDPOINT"),
		Token:    os.Getenv("ATCLOUD_TOKEN"),
	})

	err := client.DeleteObject(context.Background(), container, name)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("Object deleted:", name)
}
