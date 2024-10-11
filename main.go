package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "mycli",
	Short: "Module Loader",
	Long:  `Module Loader.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Welcome to Lilypad Module Loader!")
		loadAllowListFromImages()
	},
}

var cli, err = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
var ctx = context.Background()

//	if err != nil {
//		fmt.Println("Error creating Docker client:", err)
//		return
//	}
func export() bool {

	if err != nil {
		fmt.Println("Error creating Docker client:", err)
		return true
	}

	images, err := cli.ImageList(ctx, image.ListOptions{})
	if err != nil {
		fmt.Println("Error listing Docker images:", err)
		return true
	}

	for _, image := range images {
		for _, tag := range image.RepoTags {

			saveImage(ctx, tag, cli)
			continue
		}
	}
	return false
}

func saveImage(ctx context.Context, tag string, cli *client.Client) {
	sanitizedTag := strings.ReplaceAll(tag, "/", "_")
	outputFile := fmt.Sprintf("images/%s.tar", sanitizedTag)

	if _, err := os.Stat(outputFile); err == nil {
		fmt.Println("File already exists, skipping:", outputFile)

	}

	file, err := os.Create(outputFile)
	if err != nil {
		fmt.Println("Error creating output file:", err)

	}
	defer file.Close()

	reader, err := cli.ImageSave(ctx, []string{tag})
	if err != nil {
		fmt.Println("Error saving Docker image:", err)

	}
	defer reader.Close()

	_, err = io.Copy(file, reader)
	if err != nil {
		fmt.Println("Error writing image to file:", err)

	}

	fmt.Println("Docker image saved successfully to", outputFile)
}
func loadImage(ctx context.Context, tag string, cli *client.Client) {
	sanitizedTag := strings.ReplaceAll(tag, "/", "_")
	outputFile := fmt.Sprintf("images/%s.tar", sanitizedTag)

	if _, err := os.Stat(sanitizedTag); err == nil {
		fmt.Println("File already exists, skipping:", outputFile)

	} else {
		cmd := exec.Command("docker", "pull", tag)
		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("Error pulling image %s: %s\n", outputFile, err)
		} else {
			fmt.Printf("Successfully pulling image %s: %s\n", outputFile, output)
		}
		saveImage(ctx, tag, cli)
		// return
	}

	cmd := exec.Command("docker", "load", "-i", outputFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error loading image %s: %s\n", outputFile, err)
	} else {
		fmt.Printf("Successfully loaded image %s: %s\n", outputFile, output)
	}

	// file, err := os.Create(outputFile)
	// if err != nil {
	// 	fmt.Println("Error creating output file:", err)

	// }
	// defer file.Close()

	// loadImage(ctx, tag, cli)

	// _, err = io.Copy(file, reader)
	// if err != nil {
	// 	fmt.Println("Error writing image to file:", err)

	// }

	// fmt.Println("Docker image loaded successfully to", outputFile)
}

type ModuleB struct {
	ModuleId string
	Image    string
	Cid      string
}

func cache(data []ModuleB, useIpfs bool) {
	fmt.Println(data)
	for _, module := range data {
		if strings.HasPrefix(module.ModuleId, "http") {
			moduleUrl := strings.Replace(module.ModuleId, "github.com", "raw.githubusercontent.com", 1) + "/main/lilypad_module.json.tmpl"
			response, err := http.Get(moduleUrl)
			if err != nil {
				fmt.Println("Error fetching module URL:", err)
				continue
			}
			defer response.Body.Close()
			body, err := ioutil.ReadAll(response.Body)
			if err != nil {
				fmt.Println("Error reading response body:", err)
				continue
			}
			re := regexp.MustCompile(`"Image":\s*"([^"]+)"`)
			matches := re.FindStringSubmatch(string(body))
			if len(matches) > 1 {
				cleanedJsonString := matches[1]
				module.Image = cleanedJsonString
				fmt.Println(cleanedJsonString)
				if useIpfs && module.Cid != "" {
					fmt.Println("ipfs get " + module.Cid + "; docker load -i " + module.Cid)
				} else {
					fmt.Println("docker pull " + cleanedJsonString)
				}
			} else {
				fmt.Println("No image found in response:", string(body))
			}
		}
	}
}

type Module struct {
	ModuleId string `json:"ModuleId"`
	Image    string `json:"Image"`
}

func saveAllowListToImages() {
	allLilypadURL := "https://raw.githubusercontent.com/arsen3d/module-allowlist/main/allowlist.json"
	response, err := http.Get(allLilypadURL)
	if err != nil {
		fmt.Println("Error fetching allowlist:", err)
		return
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return
	}

	var modules []Module
	err = json.Unmarshal(body, &modules)
	if err != nil {
		fmt.Println("Error unmarshalling JSON:", err)
		return
	}

	for _, module := range modules {
		if strings.HasPrefix(module.ModuleId, "http") {
			moduleUrl := strings.Replace(module.ModuleId, "github.com", "raw.githubusercontent.com", 1) + "/main/lilypad_module.json.tmpl"
			responseModule, err := http.Get(moduleUrl)
			if err != nil {
				fmt.Println("Error fetching module:", err)
				continue
			}
			defer responseModule.Body.Close()

			moduleBody, err := ioutil.ReadAll(responseModule.Body)
			if err != nil {
				fmt.Println("Error reading module response body:", err)
				continue
			}

			re := regexp.MustCompile(`"Image":\s*"([^"]+)"`)
			matches := re.FindStringSubmatch(string(moduleBody))
			if len(matches) > 1 {
				module.Image = matches[1]
				fmt.Println(matches[1])
				cmd := exec.Command("docker", "pull", module.Image)
				output, err := cmd.CombinedOutput()
				if err != nil {
					fmt.Printf("Error pulling image %s: %s\n", module.Image, err)
				} else {
					fmt.Printf("Successfully pulled image %s: %s\n", module.Image, output)
				}
				saveImage(ctx, module.Image, cli)
			} else {
				fmt.Println("Error parsing module JSON")
			}
		}
	}

	fmt.Println(modules)
}
func loadAllowListFromImages() {
	allLilypadURL := "https://raw.githubusercontent.com/arsen3d/module-allowlist/main/allowlist.json"
	response, err := http.Get(allLilypadURL)
	if err != nil {
		fmt.Println("Error fetching allowlist:", err)
		return
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return
	}

	var modules []Module
	err = json.Unmarshal(body, &modules)
	if err != nil {
		fmt.Println("Error unmarshalling JSON:", err)
		return
	}

	for _, module := range modules {
		if strings.HasPrefix(module.ModuleId, "http") {
			moduleUrl := strings.Replace(module.ModuleId, "github.com", "raw.githubusercontent.com", 1) + "/main/lilypad_module.json.tmpl"
			responseModule, err := http.Get(moduleUrl)
			if err != nil {
				fmt.Println("Error fetching module:", err)
				continue
			}
			defer responseModule.Body.Close()

			moduleBody, err := ioutil.ReadAll(responseModule.Body)
			if err != nil {
				fmt.Println("Error reading module response body:", err)
				continue
			}

			re := regexp.MustCompile(`"Image":\s*"([^"]+)"`)
			matches := re.FindStringSubmatch(string(moduleBody))
			if len(matches) > 1 {
				module.Image = matches[1]
				fmt.Println(matches[1])

				loadImage(ctx, module.Image, cli)
			} else {
				fmt.Println("Error parsing module JSON")
			}
		}
	}

	fmt.Println(modules)
}
func pullImage(image string) {
	url := "http://host.docker.internal:2375/images/create?fromImage=" + image
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error pulling image:", err)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return
	}

	if resp.StatusCode == http.StatusOK {
		fmt.Printf("Successfully pulled image %s: %s\n", image, body)
	} else {
		fmt.Printf("Failed to pull image %s: %s\n", image, body)
	}
}
func main() {
	// Create a Docker client
	// List all Docker images
	// Loop through each image and save it
	// Sanitize the output file name
	// Check if the file already exists
	// Open the output file
	// Save the Docker image
	// Write the image to the file
	// shouldReturn := export()
	// if shouldReturn {
	// 	return
	// }
	// saveAllowListToImages()
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
