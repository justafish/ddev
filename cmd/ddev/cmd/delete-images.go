package cmd

import (
	"github.com/drud/ddev/pkg/ddevapp"
	"os"
	"sort"
	"strings"

	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/nodeps"
	"github.com/drud/ddev/pkg/util"
	"github.com/drud/ddev/pkg/version"
	docker "github.com/fsouza/go-dockerclient"
	"github.com/spf13/cobra"
)

// noConfirm: If true, --yes, we won't stop and prompt before each deletion
var deleteImagesNocConfirm bool

// deleteAllImages: If set, deletes all images created by ddev
var deleteAllImages bool

// DeleteImagesCmd implements the ddev delete images command
var DeleteImagesCmd = &cobra.Command{
	Use:   "images",
	Short: "Deletes ddev docker images not in use by current ddev version",
	Long:  "with --all it deletes all ddev docker images",
	Example: `ddev delete images
ddev delete images -y
ddev delete images --all`,

	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if !deleteImagesNocConfirm {
			if !util.Confirm("Deleting unused ddev images. \nThis is a non-destructive operation, \nbut it may require that the images be downloaded again when you need them. \nOK to continue?") {
				os.Exit(1)
			}
		}
		util.Success("Powering off ddev to avoid conflicts")
		ddevapp.PowerOff()

		client := dockerutil.GetDockerClient()

		images, err := client.ListImages(docker.ListImagesOptions{
			All: true,
		})
		if err != nil {
			util.Failed("Failed to list images: %v", err)
		}

		// The user can select to delete all ddev images.
		if deleteAllImages {
			// Attempt to find ddev images by tag, searching for "drud/ddev-".
			// Some ddev images will not be found by this tag, future work will
			// be done to improve finding database images.
			for _, image := range images {
				for _, tag := range image.RepoTags {
					if strings.HasPrefix(tag, "drud/ddev-") {
						if err = dockerutil.RemoveImage(tag); err != nil {
							util.Warning("Failed to remove %s: %v", tag, err)
						}
					}
				}
			}
			util.Success("All ddev images discovered were deleted.")
			os.Exit(0)
		}

		// Sort so that images that have -built on the end
		// come up before their parent images that don't
		sort.Slice(images, func(i, j int) bool {
			if images[i].RepoTags == nil || images[j].RepoTags == nil {
				return false
			}
			return images[i].RepoTags[0] > images[j].RepoTags[0]
		})

		webimg := version.GetWebImage()
		dbaimage := version.GetDBAImage()
		routerimage := version.RouterImage + ":" + version.RouterTag
		sshimage := version.SSHAuthImage + ":" + version.SSHAuthTag

		nameAry := strings.Split(version.GetDBImage(nodeps.MariaDB), ":")
		keepDBImageTag := "notagfound"
		if len(nameAry) > 1 {
			keepDBImageTag = nameAry[1]
		}

		// Too much code inside this loop, but complicated by multiple db images
		// and discrete names of images
		for _, image := range images {
			for _, tag := range image.RepoTags {
				// If a webimage, but doesn't match our webimage, delete it
				if strings.HasPrefix(tag, version.WebImg) && !strings.HasPrefix(tag, webimg) && !strings.HasPrefix(tag, webimg+"-built") {
					if err = dockerutil.RemoveImage(tag); err != nil {
						util.Warning("Failed to remove %s: %v", tag, err)
					}
				}
				if strings.HasPrefix(tag, "drud/ddev-dbserver") && !strings.HasSuffix(tag, keepDBImageTag) && !strings.HasSuffix(tag, keepDBImageTag+"-built") {
					if err = dockerutil.RemoveImage(tag); err != nil {
						util.Warning("Failed to remove %s: %v", tag, err)
					}
				}
				// If a dbaimage, but doesn't match our dbaimage, delete it
				if strings.HasPrefix(tag, version.DBAImg) && !strings.HasPrefix(tag, dbaimage) {
					if err = dockerutil.RemoveImage(tag); err != nil {
						util.Warning("Failed to remove %s: %v", tag, err)
					}
				}
				// If a routerImage, but doesn't match our routerimage, delete it
				if strings.HasPrefix(tag, version.RouterImage) && !strings.HasPrefix(tag, routerimage) {
					if err = dockerutil.RemoveImage(tag); err != nil {
						util.Warning("Failed to remove %s: %v", tag, err)
					}
				}
				// If a sshAgentImage, but doesn't match our sshAgentImage, delete it
				if strings.HasPrefix(tag, version.SSHAuthImage) && !strings.HasPrefix(tag, sshimage) && !strings.HasPrefix(tag, sshimage+"-built") {
					if err = dockerutil.RemoveImage(tag); err != nil {
						util.Warning("Failed to remove %s: %v", tag, err)
					}
				}
			}
		}
		util.Success("Any non-current images discovered were deleted.")
	},
}

func init() {
	DeleteImagesCmd.Flags().BoolVarP(&deleteImagesNocConfirm, "yes", "y", false, "Yes - skip confirmation prompt")
	DeleteImagesCmd.Flags().BoolVarP(&deleteAllImages, "all", "a", false, "If set, deletes all Docker images created by ddev.")
	DeleteCmd.AddCommand(DeleteImagesCmd)
}
