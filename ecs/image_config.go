package ecs

import (
	"fmt"

	"regexp"
	"strings"

	"github.com/denverdino/aliyungo/common"
	"github.com/hashicorp/packer/template/interpolate"
)

type AlicloudDiskDevice struct {
	DiskName           string `mapstructure:"disk_name"`
	DiskCategory       string `mapstructure:"disk_category"`
	DiskSize           int    `mapstructure:"disk_size"`
	SnapshotId         string `mapstructure:"disk_snapshot_id"`
	Description        string `mapstructure:"disk_description"`
	DeleteWithInstance bool   `mapstructure:"disk_delete_with_instance"`
	Device             string `mapstructure:"disk_device"`
}

type AlicloudDiskDevices struct {
	ECSSystemDiskMapping  AlicloudDiskDevice   `mapstructure:"system_disk_mapping"`
	ECSImagesDiskMappings []AlicloudDiskDevice `mapstructure:"image_disk_mappings"`
}

type AlicloudImageConfig struct {
	AlicloudImageName                 string            `mapstructure:"image_name"`
	AlicloudImageSnapshotNames                 []string            `mapstructure:"image_snapshot_names"`
	AlicloudImageVersion              string            `mapstructure:"image_version"`
	AlicloudImageDescription          string            `mapstructure:"image_description"`
	AlicloudImageShareAccounts        []string          `mapstructure:"image_share_account"`
	AlicloudImageUNShareAccounts      []string          `mapstructure:"image_unshare_account"`
	AlicloudImageDestinationRegions   []string          `mapstructure:"image_copy_regions"`
	AlicloudImageDestinationNames     []string          `mapstructure:"image_copy_names"`
	AlicloudImageDestinationSnapshotNames     map[string][]string         `mapstructure:"image_copy_snapshot_names"`
	AlicloudImageForceDelete          bool              `mapstructure:"image_force_delete"`
	AlicloudImageForceDeleteSnapshots bool              `mapstructure:"image_force_delete_snapshots"`
	AlicloudImageForceDeleteInstances bool              `mapstructure:"image_force_delete_instances"`
	AlicloudImageIgnoreDataDisks      bool              `mapstructure:"image_ignore_data_disks"`
	AlicloudImageSkipRegionValidation bool              `mapstructure:"skip_region_validation"`
	AlicloudImageTags                 map[string]string `mapstructure:"tags"`
	AlicloudDiskDevices               `mapstructure:",squash"`
}

func (c *AlicloudImageConfig) Prepare(ctx *interpolate.Context) []error {
	var errs []error
	if c.AlicloudImageName == "" {
		errs = append(errs, fmt.Errorf("image_name must be specified"))
	} else {
		if partErrs := validateImageName(c.AlicloudImageName, "image_name"); partErrs != nil {
			errs = append(errs, partErrs...)
		}
	}

	if len(c.AlicloudImageDestinationNames) > 0 {
		for index, destName := range c.AlicloudImageDestinationNames {
			if destName == "" {
				continue
			}

			if partErrs := validateImageName(destName, fmt.Sprintf("image_copy_names[%d]", index)); partErrs != nil {
				errs = append(errs, partErrs...)
			}
		}
	}

	if len(c.AlicloudImageSnapshotNames) > 0 {
		for index, snapshotName := range c.AlicloudImageSnapshotNames {
			if snapshotName == "" {
				continue
			}

			if partErrs := validateSnapshotName(snapshotName, fmt.Sprintf("image_snapshot_names[%d]", index)); partErrs != nil {
				errs = append(errs, partErrs...)
			}
		}
	}

	if len(c.AlicloudImageDestinationRegions) > 0 {
		regionSet := make(map[string]struct{})
		regions := make([]string, 0, len(c.AlicloudImageDestinationRegions))

		for _, region := range c.AlicloudImageDestinationRegions {
			// If we already saw the region, then don't look again
			if _, ok := regionSet[region]; ok {
				continue
			}

			// Mark that we saw the region
			regionSet[region] = struct{}{}

			if !c.AlicloudImageSkipRegionValidation {
				// Verify the region is real
				if err := validateRegion(region); err != nil {
					errs = append(errs, err)
					continue
				}
			}

			regions = append(regions, region)
		}

		c.AlicloudImageDestinationRegions = regions
	}

	if len(c.AlicloudImageDestinationSnapshotNames) > 0 {
		for region, snapshotNames := range c.AlicloudImageDestinationSnapshotNames {
			if !c.AlicloudImageSkipRegionValidation {
				if err := validateRegion(region); err != nil {
					errs = append(errs, err)
				}
			}

			for index, snapshotName := range snapshotNames {
				if snapshotName == "" {
					continue
				}

				if partErrs := validateSnapshotName(snapshotName, fmt.Sprintf("image_snapshot_names[%d]", index)); partErrs != nil {
					errs = append(errs, partErrs...)
				}
			}
		}
	}

	if len(errs) > 0 {
		return errs
	}

	return nil
}

func validateImageName(name string, option string) []error {
	var errs []error

	if len(name) < 2 || len(name) > 128 {
		errs = append(errs, fmt.Errorf("%s must less than 128 letters and more than 1 letters", option))
	} else if strings.HasPrefix(name, "http://") || strings.HasPrefix(name, "https://") {
		errs = append(errs, fmt.Errorf("%s can't start with 'http://' or 'https://'", option))
	}

	reg := regexp.MustCompile("\\s+")
	if reg.FindString(name) != "" {
		errs = append(errs, fmt.Errorf("%s can't include spaces", name))
	}

	if len(errs) > 0 {
		return errs
	}

	return nil
}

func validateSnapshotName(name string, option string) []error{
	errs := validateImageName(name, option)

	if strings.HasPrefix(name, "auto"){
		if errs == nil {
			errs = []error{}
		}

		errs = append(errs, fmt.Errorf("%s can't start with 'auto'", option))
	}

	if len(errs) > 0 {
		return errs
	}

	return nil
}

func validateRegion(region string) error {

	for _, valid := range common.ValidRegions {
		if region == string(valid) {
			return nil
		}
	}

	return fmt.Errorf("Not a valid alicloud region: %s", region)
}
