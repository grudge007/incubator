package images

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

const ImagePath = "/var/incubator/images/"

type Image struct {
	Version string `yaml:"version"`
	Path    string `yaml:"path"`
}
type AvailableImages struct {
	Images map[string][]Image `yaml:"images"`
}

func MapImages(imageName string) {

}

func main() {
	StoreImages()
}

func StoreImages() AvailableImages {
	data, err := os.ReadFile("/opt/incubator/internal/configs/available-images.yaml")
	if err != nil {
		panic(err)
	}
	var cfg AvailableImages
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		log.Fatal(err)
	}

	// fmt.Println(string(data))
	return cfg

}
