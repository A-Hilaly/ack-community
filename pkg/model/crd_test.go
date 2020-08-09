// Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//	 http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

package model_test

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aws/aws-controllers-k8s/pkg/model"
	"github.com/aws/aws-controllers-k8s/pkg/testutil"
)

func attrCamelNames(fields map[string]*model.CRDField) []string {
	res := []string{}
	for _, attr := range fields {
		res = append(res, attr.Names.Camel)
	}
	sort.Strings(res)
	return res
}

func getCRDByName(name string, crds []*model.CRD) *model.CRD {
	for _, c := range crds {
		if c.Names.Original == name {
			return c
		}
	}
	return nil
}

func TestSNSTopic(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	sh := testutil.NewSchemaHelperForService(t, "sns")

	crds, err := sh.GetCRDs()
	require.Nil(err)

	crd := getCRDByName("Topic", crds)
	require.NotNil(crd)

	assert.Equal("Topic", crd.Names.Camel)
	assert.Equal("topic", crd.Names.CamelLower)
	assert.Equal("topic", crd.Names.Snake)

	// The SNS Topic API is a little weird. There are Create and Delete
	// operations ("CreateTopic", "DeleteTopic") but there is no ReadOne
	// operation (there is a "GetTopicAttributes" call though) or Update
	// operation (there is a "SetTopicAttributes" call though). And there is a
	// ReadMany operation (ListTopics)
	require.NotNil(crd.Ops)

	assert.NotNil(crd.Ops.Create)
	assert.NotNil(crd.Ops.Delete)
	assert.NotNil(crd.Ops.ReadMany)
	assert.NotNil(crd.Ops.GetAttributes)
	assert.NotNil(crd.Ops.SetAttributes)

	assert.Nil(crd.Ops.ReadOne)
	assert.Nil(crd.Ops.Update)

	specFields := crd.SpecFields
	statusFields := crd.StatusFields

	// The SNS Topic uses an "Attributes" map[string]*string to masquerade
	// real fields. DeliveryPolicy, Policy, KMSMasterKeyID and DisplayName are
	// all examples of these masqueraded fields...
	expSpecFieldCamel := []string{
		"DeliveryPolicy",
		"DisplayName",
		"KMSMasterKeyID",
		"Name",
		"Policy",
		"Tags",
	}
	assert.Equal(expSpecFieldCamel, attrCamelNames(specFields))

	// The SNS Topic uses an "Attributes" map[string]*string to masquerade
	// real fields. EffectiveDeliveryPolicy and Owner are
	// examples of these masqueraded fields that are ReadOnly and thus belong
	// in the Status struct
	expStatusFieldCamel := []string{
		// "TopicARN" is in the output shape for CreateTopic, but it is removed
		// because it is the primary resource object's ARN field and the
		// SDKMapper has identified it as the source for the standard
		// Status.ACKResourceMetadata.ARN field
		"EffectiveDeliveryPolicy",
		"Owner",
	}
	assert.Equal(expStatusFieldCamel, attrCamelNames(statusFields))

	// The input shape for the Create operation is set from a variety of scalar
	// and non-scalar types and the SNS API features an Attributes parameter
	// that is actually a map[string]*string of real field values that are
	// unpacked by the code generator.
	expCreateInput := `
	attrMap := map[string]*string{}
	if r.ko.Spec.DeliveryPolicy != nil {
		attrMap["DeliveryPolicy"] = r.ko.Spec.DeliveryPolicy
	}
	if r.ko.Spec.DisplayName != nil {
		attrMap["DisplayName"] = r.ko.Spec.DisplayName
	}
	if r.ko.Spec.KMSMasterKeyID != nil {
		attrMap["KmsMasterKeyId"] = r.ko.Spec.KMSMasterKeyID
	}
	if r.ko.Spec.Policy != nil {
		attrMap["Policy"] = r.ko.Spec.Policy
	}
	res.SetAttributes(attrMap)
	if r.ko.Spec.Name != nil {
		res.SetName(*r.ko.Spec.Name)
	}
	if r.ko.Spec.Tags != nil {
		f2 := []*svcsdk.Tag{}
		for _, f2iter := range r.ko.Spec.Tags {
			f2elem := &svcsdk.Tag{}
			if f2iter.Key != nil {
				f2elem.SetKey(*f2iter.Key)
			}
			if f2iter.Value != nil {
				f2elem.SetValue(*f2iter.Value)
			}
			f2 = append(f2, f2elem)
		}
		res.SetTags(f2)
	}
`
	assert.Equal(expCreateInput, crd.GoCodeSetInput(model.OpTypeCreate, "r.ko", "res", 1))

	// None of the fields in the Topic resource's CreateTopicInput shape are
	// returned in the CreateTopicOutput shape, so none of them return any Go
	// code for setting a Status struct field to a corresponding Create Output
	// Shape member
	expCreateOutput := `
`
	assert.Equal(expCreateOutput, crd.GoCodeSetOutput(model.OpTypeCreate, "resp", "ko.Status", 1))

	// The input shape for the GetAttributes operation has a single TopicArn
	// field. This field represents the ARN of the primary resource (the Topic
	// itself) and should be set specially from the ACKResourceMetadata.ARN
	// field in the TopicStatus struct
	expGetAttrsInput := `
	if r.ko.Status.ACKResourceMetadata != nil && r.ko.Status.ACKResourceMetadata.ARN != nil {
		res.SetTopicArn(string(*r.ko.Status.ACKResourceMetadata.ARN))
	}
`
	assert.Equal(expGetAttrsInput, crd.GoCodeGetAttributesSetInput("r.ko", "res", 1))

	// The output shape for the GetAttributes operation contains a single field
	// "Attributes" that must be unpacked into the Topic CRD's Status fields.
	// There are only three attribute keys that are *not* in the Input shape
	// (and thus in the Spec fields). Two of them are the tesource's ARN and
	// AWS Owner account ID, both of which are handled specially.
	expGetAttrsOutput := `
	ko.Status.EffectiveDeliveryPolicy = resp.Attributes["EffectiveDeliveryPolicy"]
	tmpOwnerID := ackv1alpha1.AWSAccountID(*resp.Attributes["Owner"])
	ko.Status.ACKResourceMetadata.OwnerAccountID = &tmpOwnerID
	tmpARN := ackv1alpha1.AWSResourceName(*resp.Attributes["TopicArn"])
	ko.Status.ACKResourceMetadata.ARN = &tmpARN
`
	assert.Equal(expGetAttrsOutput, crd.GoCodeGetAttributesSetOutput("resp", "ko.Status", 1))
}

func TestEC2LaunchTemplate(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	sh := testutil.NewSchemaHelperForService(t, "ec2")

	crds, err := sh.GetCRDs()
	require.Nil(err)

	crd := getCRDByName("LaunchTemplate", crds)
	require.NotNil(crd)

	assert.Equal("LaunchTemplate", crd.Names.Camel)
	assert.Equal("launchTemplate", crd.Names.CamelLower)
	assert.Equal("launch_template", crd.Names.Snake)

	specFields := crd.SpecFields
	statusFields := crd.StatusFields

	expSpecFieldCamel := []string{
		// TODO(jaypipes): DryRun and ClientToken are examples of two fields in
		// the resource input shape that need to be stripped out of the CRD. We
		// need to instruct the code generator that these types of fields are
		// not germane to the resource itself...
		"ClientToken",
		"DryRun",
		"LaunchTemplateData",
		"LaunchTemplateName",
		// TODO(jaypipes): Here's an example of where we need to instruct the
		// code generator to rename the "TagSpecifications" field to simply
		// "Tags" and place it into the common Spec.Tags field.
		"TagSpecifications",
		"VersionDescription",
	}
	assert.Equal(expSpecFieldCamel, attrCamelNames(specFields))

	expStatusFieldCamel := []string{
		"CreateTime",
		"CreatedBy",
		"DefaultVersionNumber",
		"LatestVersionNumber",
		// TODO(jaypipes): Handle "Id" Fields like "LaunchTemplateId" in the
		// same way as we handle ARN-ified modern service APIs and use the
		// SDKMapper to instruct the code generator that this field represents
		// the primary resource object's identifier field.
		"LaunchTemplateID",
		// LaunchTemplateName excluded because it matches input shape.,
		// TODO(jaypipes): Tags field should be excluded because it is the same
		// as the input shape's "TagSpecifications" field...
		"Tags",
	}
	assert.Equal(expStatusFieldCamel, attrCamelNames(statusFields))

	// LaunchTemplateName is in the LaunchTemplate resource's CreateTopicInput shape and also
	// returned in the CreateLaunchTemplateResult shape, so it should have
	// Go code to set the Input Shape member from the Spec field but not set a
	// Status field from the Create Output Shape member
	expCreateInput := `
	if r.ko.Spec.ClientToken != nil {
		res.SetClientToken(*r.ko.Spec.ClientToken)
	}
	if r.ko.Spec.DryRun != nil {
		res.SetDryRun(*r.ko.Spec.DryRun)
	}
	if r.ko.Spec.LaunchTemplateData != nil {
		f2 := &svcsdk.RequestLaunchTemplateData{}
		if r.ko.Spec.LaunchTemplateData.BlockDeviceMappings != nil {
			f2f0 := []*svcsdk.LaunchTemplateBlockDeviceMappingRequest{}
			for _, f2f0iter := range r.ko.Spec.LaunchTemplateData.BlockDeviceMappings {
				f2f0elem := &svcsdk.LaunchTemplateBlockDeviceMappingRequest{}
				if f2f0iter.DeviceName != nil {
					f2f0elem.SetDeviceName(*f2f0iter.DeviceName)
				}
				if f2f0iter.EBS != nil {
					f2f0elemf1 := &svcsdk.LaunchTemplateEbsBlockDeviceRequest{}
					if f2f0iter.EBS.DeleteOnTermination != nil {
						f2f0elemf1.SetDeleteOnTermination(*f2f0iter.EBS.DeleteOnTermination)
					}
					if f2f0iter.EBS.Encrypted != nil {
						f2f0elemf1.SetEncrypted(*f2f0iter.EBS.Encrypted)
					}
					if f2f0iter.EBS.IOPS != nil {
						f2f0elemf1.SetIops(*f2f0iter.EBS.IOPS)
					}
					if f2f0iter.EBS.KMSKeyID != nil {
						f2f0elemf1.SetKmsKeyId(*f2f0iter.EBS.KMSKeyID)
					}
					if f2f0iter.EBS.SnapshotID != nil {
						f2f0elemf1.SetSnapshotId(*f2f0iter.EBS.SnapshotID)
					}
					if f2f0iter.EBS.VolumeSize != nil {
						f2f0elemf1.SetVolumeSize(*f2f0iter.EBS.VolumeSize)
					}
					if f2f0iter.EBS.VolumeType != nil {
						f2f0elemf1.SetVolumeType(*f2f0iter.EBS.VolumeType)
					}
					f2f0elem.SetEbs(f2f0elemf1)
				}
				if f2f0iter.NoDevice != nil {
					f2f0elem.SetNoDevice(*f2f0iter.NoDevice)
				}
				if f2f0iter.VirtualName != nil {
					f2f0elem.SetVirtualName(*f2f0iter.VirtualName)
				}
				f2f0 = append(f2f0, f2f0elem)
			}
			f2.SetBlockDeviceMappings(f2f0)
		}
		if r.ko.Spec.LaunchTemplateData.CapacityReservationSpecification != nil {
			f2f1 := &svcsdk.LaunchTemplateCapacityReservationSpecificationRequest{}
			if r.ko.Spec.LaunchTemplateData.CapacityReservationSpecification.CapacityReservationPreference != nil {
				f2f1.SetCapacityReservationPreference(*r.ko.Spec.LaunchTemplateData.CapacityReservationSpecification.CapacityReservationPreference)
			}
			if r.ko.Spec.LaunchTemplateData.CapacityReservationSpecification.CapacityReservationTarget != nil {
				f2f1f1 := &svcsdk.CapacityReservationTarget{}
				if r.ko.Spec.LaunchTemplateData.CapacityReservationSpecification.CapacityReservationTarget.CapacityReservationID != nil {
					f2f1f1.SetCapacityReservationId(*r.ko.Spec.LaunchTemplateData.CapacityReservationSpecification.CapacityReservationTarget.CapacityReservationID)
				}
				f2f1.SetCapacityReservationTarget(f2f1f1)
			}
			f2.SetCapacityReservationSpecification(f2f1)
		}
		if r.ko.Spec.LaunchTemplateData.CPUOptions != nil {
			f2f2 := &svcsdk.LaunchTemplateCpuOptionsRequest{}
			if r.ko.Spec.LaunchTemplateData.CPUOptions.CoreCount != nil {
				f2f2.SetCoreCount(*r.ko.Spec.LaunchTemplateData.CPUOptions.CoreCount)
			}
			if r.ko.Spec.LaunchTemplateData.CPUOptions.ThreadsPerCore != nil {
				f2f2.SetThreadsPerCore(*r.ko.Spec.LaunchTemplateData.CPUOptions.ThreadsPerCore)
			}
			f2.SetCpuOptions(f2f2)
		}
		if r.ko.Spec.LaunchTemplateData.CreditSpecification != nil {
			f2f3 := &svcsdk.CreditSpecificationRequest{}
			if r.ko.Spec.LaunchTemplateData.CreditSpecification.CPUCredits != nil {
				f2f3.SetCpuCredits(*r.ko.Spec.LaunchTemplateData.CreditSpecification.CPUCredits)
			}
			f2.SetCreditSpecification(f2f3)
		}
		if r.ko.Spec.LaunchTemplateData.DisableAPITermination != nil {
			f2.SetDisableApiTermination(*r.ko.Spec.LaunchTemplateData.DisableAPITermination)
		}
		if r.ko.Spec.LaunchTemplateData.EBSOptimized != nil {
			f2.SetEbsOptimized(*r.ko.Spec.LaunchTemplateData.EBSOptimized)
		}
		if r.ko.Spec.LaunchTemplateData.ElasticGPUSpecifications != nil {
			f2f6 := []*svcsdk.ElasticGpuSpecification{}
			for _, f2f6iter := range r.ko.Spec.LaunchTemplateData.ElasticGPUSpecifications {
				f2f6elem := &svcsdk.ElasticGpuSpecification{}
				if f2f6iter.Type != nil {
					f2f6elem.SetType(*f2f6iter.Type)
				}
				f2f6 = append(f2f6, f2f6elem)
			}
			f2.SetElasticGpuSpecifications(f2f6)
		}
		if r.ko.Spec.LaunchTemplateData.ElasticInferenceAccelerators != nil {
			f2f7 := []*svcsdk.LaunchTemplateElasticInferenceAccelerator{}
			for _, f2f7iter := range r.ko.Spec.LaunchTemplateData.ElasticInferenceAccelerators {
				f2f7elem := &svcsdk.LaunchTemplateElasticInferenceAccelerator{}
				if f2f7iter.Count != nil {
					f2f7elem.SetCount(*f2f7iter.Count)
				}
				if f2f7iter.Type != nil {
					f2f7elem.SetType(*f2f7iter.Type)
				}
				f2f7 = append(f2f7, f2f7elem)
			}
			f2.SetElasticInferenceAccelerators(f2f7)
		}
		if r.ko.Spec.LaunchTemplateData.HibernationOptions != nil {
			f2f8 := &svcsdk.LaunchTemplateHibernationOptionsRequest{}
			if r.ko.Spec.LaunchTemplateData.HibernationOptions.Configured != nil {
				f2f8.SetConfigured(*r.ko.Spec.LaunchTemplateData.HibernationOptions.Configured)
			}
			f2.SetHibernationOptions(f2f8)
		}
		if r.ko.Spec.LaunchTemplateData.IAMInstanceProfile != nil {
			f2f9 := &svcsdk.LaunchTemplateIamInstanceProfileSpecificationRequest{}
			if r.ko.Spec.LaunchTemplateData.IAMInstanceProfile.ARN != nil {
				f2f9.SetArn(*r.ko.Spec.LaunchTemplateData.IAMInstanceProfile.ARN)
			}
			if r.ko.Spec.LaunchTemplateData.IAMInstanceProfile.Name != nil {
				f2f9.SetName(*r.ko.Spec.LaunchTemplateData.IAMInstanceProfile.Name)
			}
			f2.SetIamInstanceProfile(f2f9)
		}
		if r.ko.Spec.LaunchTemplateData.ImageID != nil {
			f2.SetImageId(*r.ko.Spec.LaunchTemplateData.ImageID)
		}
		if r.ko.Spec.LaunchTemplateData.InstanceInitiatedShutdownBehavior != nil {
			f2.SetInstanceInitiatedShutdownBehavior(*r.ko.Spec.LaunchTemplateData.InstanceInitiatedShutdownBehavior)
		}
		if r.ko.Spec.LaunchTemplateData.InstanceMarketOptions != nil {
			f2f12 := &svcsdk.LaunchTemplateInstanceMarketOptionsRequest{}
			if r.ko.Spec.LaunchTemplateData.InstanceMarketOptions.MarketType != nil {
				f2f12.SetMarketType(*r.ko.Spec.LaunchTemplateData.InstanceMarketOptions.MarketType)
			}
			if r.ko.Spec.LaunchTemplateData.InstanceMarketOptions.SpotOptions != nil {
				f2f12f1 := &svcsdk.LaunchTemplateSpotMarketOptionsRequest{}
				if r.ko.Spec.LaunchTemplateData.InstanceMarketOptions.SpotOptions.BlockDurationMinutes != nil {
					f2f12f1.SetBlockDurationMinutes(*r.ko.Spec.LaunchTemplateData.InstanceMarketOptions.SpotOptions.BlockDurationMinutes)
				}
				if r.ko.Spec.LaunchTemplateData.InstanceMarketOptions.SpotOptions.InstanceInterruptionBehavior != nil {
					f2f12f1.SetInstanceInterruptionBehavior(*r.ko.Spec.LaunchTemplateData.InstanceMarketOptions.SpotOptions.InstanceInterruptionBehavior)
				}
				if r.ko.Spec.LaunchTemplateData.InstanceMarketOptions.SpotOptions.MaxPrice != nil {
					f2f12f1.SetMaxPrice(*r.ko.Spec.LaunchTemplateData.InstanceMarketOptions.SpotOptions.MaxPrice)
				}
				if r.ko.Spec.LaunchTemplateData.InstanceMarketOptions.SpotOptions.SpotInstanceType != nil {
					f2f12f1.SetSpotInstanceType(*r.ko.Spec.LaunchTemplateData.InstanceMarketOptions.SpotOptions.SpotInstanceType)
				}
				if r.ko.Spec.LaunchTemplateData.InstanceMarketOptions.SpotOptions.ValidUntil != nil {
					f2f12f1.SetValidUntil(r.ko.Spec.LaunchTemplateData.InstanceMarketOptions.SpotOptions.ValidUntil.Time)
				}
				f2f12.SetSpotOptions(f2f12f1)
			}
			f2.SetInstanceMarketOptions(f2f12)
		}
		if r.ko.Spec.LaunchTemplateData.InstanceType != nil {
			f2.SetInstanceType(*r.ko.Spec.LaunchTemplateData.InstanceType)
		}
		if r.ko.Spec.LaunchTemplateData.KernelID != nil {
			f2.SetKernelId(*r.ko.Spec.LaunchTemplateData.KernelID)
		}
		if r.ko.Spec.LaunchTemplateData.KeyName != nil {
			f2.SetKeyName(*r.ko.Spec.LaunchTemplateData.KeyName)
		}
		if r.ko.Spec.LaunchTemplateData.LicenseSpecifications != nil {
			f2f16 := []*svcsdk.LaunchTemplateLicenseConfigurationRequest{}
			for _, f2f16iter := range r.ko.Spec.LaunchTemplateData.LicenseSpecifications {
				f2f16elem := &svcsdk.LaunchTemplateLicenseConfigurationRequest{}
				if f2f16iter.LicenseConfigurationARN != nil {
					f2f16elem.SetLicenseConfigurationArn(*f2f16iter.LicenseConfigurationARN)
				}
				f2f16 = append(f2f16, f2f16elem)
			}
			f2.SetLicenseSpecifications(f2f16)
		}
		if r.ko.Spec.LaunchTemplateData.MetadataOptions != nil {
			f2f17 := &svcsdk.LaunchTemplateInstanceMetadataOptionsRequest{}
			if r.ko.Spec.LaunchTemplateData.MetadataOptions.HTTPEndpoint != nil {
				f2f17.SetHttpEndpoint(*r.ko.Spec.LaunchTemplateData.MetadataOptions.HTTPEndpoint)
			}
			if r.ko.Spec.LaunchTemplateData.MetadataOptions.HTTPPutResponseHopLimit != nil {
				f2f17.SetHttpPutResponseHopLimit(*r.ko.Spec.LaunchTemplateData.MetadataOptions.HTTPPutResponseHopLimit)
			}
			if r.ko.Spec.LaunchTemplateData.MetadataOptions.HTTPTokens != nil {
				f2f17.SetHttpTokens(*r.ko.Spec.LaunchTemplateData.MetadataOptions.HTTPTokens)
			}
			f2.SetMetadataOptions(f2f17)
		}
		if r.ko.Spec.LaunchTemplateData.Monitoring != nil {
			f2f18 := &svcsdk.LaunchTemplatesMonitoringRequest{}
			if r.ko.Spec.LaunchTemplateData.Monitoring.Enabled != nil {
				f2f18.SetEnabled(*r.ko.Spec.LaunchTemplateData.Monitoring.Enabled)
			}
			f2.SetMonitoring(f2f18)
		}
		if r.ko.Spec.LaunchTemplateData.NetworkInterfaces != nil {
			f2f19 := []*svcsdk.LaunchTemplateInstanceNetworkInterfaceSpecificationRequest{}
			for _, f2f19iter := range r.ko.Spec.LaunchTemplateData.NetworkInterfaces {
				f2f19elem := &svcsdk.LaunchTemplateInstanceNetworkInterfaceSpecificationRequest{}
				if f2f19iter.AssociatePublicIPAddress != nil {
					f2f19elem.SetAssociatePublicIpAddress(*f2f19iter.AssociatePublicIPAddress)
				}
				if f2f19iter.DeleteOnTermination != nil {
					f2f19elem.SetDeleteOnTermination(*f2f19iter.DeleteOnTermination)
				}
				if f2f19iter.Description != nil {
					f2f19elem.SetDescription(*f2f19iter.Description)
				}
				if f2f19iter.DeviceIndex != nil {
					f2f19elem.SetDeviceIndex(*f2f19iter.DeviceIndex)
				}
				if f2f19iter.Groups != nil {
					f2f19elemf4 := []*string{}
					for _, f2f19elemf4iter := range f2f19iter.Groups {
						var f2f19elemf4elem string
						f2f19elemf4elem = *f2f19elemf4iter
						f2f19elemf4 = append(f2f19elemf4, &f2f19elemf4elem)
					}
					f2f19elem.SetGroups(f2f19elemf4)
				}
				if f2f19iter.InterfaceType != nil {
					f2f19elem.SetInterfaceType(*f2f19iter.InterfaceType)
				}
				if f2f19iter.IPv6AddressCount != nil {
					f2f19elem.SetIpv6AddressCount(*f2f19iter.IPv6AddressCount)
				}
				if f2f19iter.IPv6Addresses != nil {
					f2f19elemf7 := []*svcsdk.InstanceIpv6AddressRequest{}
					for _, f2f19elemf7iter := range f2f19iter.IPv6Addresses {
						f2f19elemf7elem := &svcsdk.InstanceIpv6AddressRequest{}
						if f2f19elemf7iter.IPv6Address != nil {
							f2f19elemf7elem.SetIpv6Address(*f2f19elemf7iter.IPv6Address)
						}
						f2f19elemf7 = append(f2f19elemf7, f2f19elemf7elem)
					}
					f2f19elem.SetIpv6Addresses(f2f19elemf7)
				}
				if f2f19iter.NetworkInterfaceID != nil {
					f2f19elem.SetNetworkInterfaceId(*f2f19iter.NetworkInterfaceID)
				}
				if f2f19iter.PrivateIPAddress != nil {
					f2f19elem.SetPrivateIpAddress(*f2f19iter.PrivateIPAddress)
				}
				if f2f19iter.PrivateIPAddresses != nil {
					f2f19elemf10 := []*svcsdk.PrivateIpAddressSpecification{}
					for _, f2f19elemf10iter := range f2f19iter.PrivateIPAddresses {
						f2f19elemf10elem := &svcsdk.PrivateIpAddressSpecification{}
						if f2f19elemf10iter.Primary != nil {
							f2f19elemf10elem.SetPrimary(*f2f19elemf10iter.Primary)
						}
						if f2f19elemf10iter.PrivateIPAddress != nil {
							f2f19elemf10elem.SetPrivateIpAddress(*f2f19elemf10iter.PrivateIPAddress)
						}
						f2f19elemf10 = append(f2f19elemf10, f2f19elemf10elem)
					}
					f2f19elem.SetPrivateIpAddresses(f2f19elemf10)
				}
				if f2f19iter.SecondaryPrivateIPAddressCount != nil {
					f2f19elem.SetSecondaryPrivateIpAddressCount(*f2f19iter.SecondaryPrivateIPAddressCount)
				}
				if f2f19iter.SubnetID != nil {
					f2f19elem.SetSubnetId(*f2f19iter.SubnetID)
				}
				f2f19 = append(f2f19, f2f19elem)
			}
			f2.SetNetworkInterfaces(f2f19)
		}
		if r.ko.Spec.LaunchTemplateData.Placement != nil {
			f2f20 := &svcsdk.LaunchTemplatePlacementRequest{}
			if r.ko.Spec.LaunchTemplateData.Placement.Affinity != nil {
				f2f20.SetAffinity(*r.ko.Spec.LaunchTemplateData.Placement.Affinity)
			}
			if r.ko.Spec.LaunchTemplateData.Placement.AvailabilityZone != nil {
				f2f20.SetAvailabilityZone(*r.ko.Spec.LaunchTemplateData.Placement.AvailabilityZone)
			}
			if r.ko.Spec.LaunchTemplateData.Placement.GroupName != nil {
				f2f20.SetGroupName(*r.ko.Spec.LaunchTemplateData.Placement.GroupName)
			}
			if r.ko.Spec.LaunchTemplateData.Placement.HostID != nil {
				f2f20.SetHostId(*r.ko.Spec.LaunchTemplateData.Placement.HostID)
			}
			if r.ko.Spec.LaunchTemplateData.Placement.HostResourceGroupARN != nil {
				f2f20.SetHostResourceGroupArn(*r.ko.Spec.LaunchTemplateData.Placement.HostResourceGroupARN)
			}
			if r.ko.Spec.LaunchTemplateData.Placement.PartitionNumber != nil {
				f2f20.SetPartitionNumber(*r.ko.Spec.LaunchTemplateData.Placement.PartitionNumber)
			}
			if r.ko.Spec.LaunchTemplateData.Placement.SpreadDomain != nil {
				f2f20.SetSpreadDomain(*r.ko.Spec.LaunchTemplateData.Placement.SpreadDomain)
			}
			if r.ko.Spec.LaunchTemplateData.Placement.Tenancy != nil {
				f2f20.SetTenancy(*r.ko.Spec.LaunchTemplateData.Placement.Tenancy)
			}
			f2.SetPlacement(f2f20)
		}
		if r.ko.Spec.LaunchTemplateData.RamDiskID != nil {
			f2.SetRamDiskId(*r.ko.Spec.LaunchTemplateData.RamDiskID)
		}
		if r.ko.Spec.LaunchTemplateData.SecurityGroupIDs != nil {
			f2f22 := []*string{}
			for _, f2f22iter := range r.ko.Spec.LaunchTemplateData.SecurityGroupIDs {
				var f2f22elem string
				f2f22elem = *f2f22iter
				f2f22 = append(f2f22, &f2f22elem)
			}
			f2.SetSecurityGroupIds(f2f22)
		}
		if r.ko.Spec.LaunchTemplateData.SecurityGroups != nil {
			f2f23 := []*string{}
			for _, f2f23iter := range r.ko.Spec.LaunchTemplateData.SecurityGroups {
				var f2f23elem string
				f2f23elem = *f2f23iter
				f2f23 = append(f2f23, &f2f23elem)
			}
			f2.SetSecurityGroups(f2f23)
		}
		if r.ko.Spec.LaunchTemplateData.TagSpecifications != nil {
			f2f24 := []*svcsdk.LaunchTemplateTagSpecificationRequest{}
			for _, f2f24iter := range r.ko.Spec.LaunchTemplateData.TagSpecifications {
				f2f24elem := &svcsdk.LaunchTemplateTagSpecificationRequest{}
				if f2f24iter.ResourceType != nil {
					f2f24elem.SetResourceType(*f2f24iter.ResourceType)
				}
				if f2f24iter.Tags != nil {
					f2f24elemf1 := []*svcsdk.Tag{}
					for _, f2f24elemf1iter := range f2f24iter.Tags {
						f2f24elemf1elem := &svcsdk.Tag{}
						if f2f24elemf1iter.Key != nil {
							f2f24elemf1elem.SetKey(*f2f24elemf1iter.Key)
						}
						if f2f24elemf1iter.Value != nil {
							f2f24elemf1elem.SetValue(*f2f24elemf1iter.Value)
						}
						f2f24elemf1 = append(f2f24elemf1, f2f24elemf1elem)
					}
					f2f24elem.SetTags(f2f24elemf1)
				}
				f2f24 = append(f2f24, f2f24elem)
			}
			f2.SetTagSpecifications(f2f24)
		}
		if r.ko.Spec.LaunchTemplateData.UserData != nil {
			f2.SetUserData(*r.ko.Spec.LaunchTemplateData.UserData)
		}
		res.SetLaunchTemplateData(f2)
	}
	if r.ko.Spec.LaunchTemplateName != nil {
		res.SetLaunchTemplateName(*r.ko.Spec.LaunchTemplateName)
	}
	if r.ko.Spec.TagSpecifications != nil {
		f4 := []*svcsdk.TagSpecification{}
		for _, f4iter := range r.ko.Spec.TagSpecifications {
			f4elem := &svcsdk.TagSpecification{}
			if f4iter.ResourceType != nil {
				f4elem.SetResourceType(*f4iter.ResourceType)
			}
			if f4iter.Tags != nil {
				f4elemf1 := []*svcsdk.Tag{}
				for _, f4elemf1iter := range f4iter.Tags {
					f4elemf1elem := &svcsdk.Tag{}
					if f4elemf1iter.Key != nil {
						f4elemf1elem.SetKey(*f4elemf1iter.Key)
					}
					if f4elemf1iter.Value != nil {
						f4elemf1elem.SetValue(*f4elemf1iter.Value)
					}
					f4elemf1 = append(f4elemf1, f4elemf1elem)
				}
				f4elem.SetTags(f4elemf1)
			}
			f4 = append(f4, f4elem)
		}
		res.SetTagSpecifications(f4)
	}
	if r.ko.Spec.VersionDescription != nil {
		res.SetVersionDescription(*r.ko.Spec.VersionDescription)
	}
`
	assert.Equal(expCreateInput, crd.GoCodeSetInput(model.OpTypeCreate, "r.ko", "res", 1))

	// Check that we properly determined how to find the CreatedBy attribute
	// within the CreateLaunchTemplateResult shape, which has a single field called
	// "LaunchTemplate" that contains the CreatedBy field.
	expCreateOutput := `
	if resp.LaunchTemplate.CreateTime != nil {
		ko.Status.CreateTime = &metav1.Time{*resp.LaunchTemplate.CreateTime}
	}
	if resp.LaunchTemplate.CreatedBy != nil {
		ko.Status.CreatedBy = resp.LaunchTemplate.CreatedBy
	}
	if resp.LaunchTemplate.DefaultVersionNumber != nil {
		ko.Status.DefaultVersionNumber = resp.LaunchTemplate.DefaultVersionNumber
	}
	if resp.LaunchTemplate.LatestVersionNumber != nil {
		ko.Status.LatestVersionNumber = resp.LaunchTemplate.LatestVersionNumber
	}
	if resp.LaunchTemplate.LaunchTemplateId != nil {
		ko.Status.LaunchTemplateID = resp.LaunchTemplate.LaunchTemplateId
	}
	if resp.LaunchTemplate.Tags != nil {
		f6 := []*svcapitypes.Tag{}
		for _, f6iter := range resp.LaunchTemplate.Tags {
			f6elem := &svcapitypes.Tag{}
			if f6iter.Key != nil {
				f6elem.Key = f6iter.Key
			}
			if f6iter.Value != nil {
				f6elem.Value = f6iter.Value
			}
			f6 = append(f6, f6elem)
		}
		ko.Status.Tags = f6
	}
`
	assert.Equal(expCreateOutput, crd.GoCodeSetOutput(model.OpTypeCreate, "resp", "ko.Status", 1))

	// The EC2 LaunchTemplate API has a "normal" set of CUD operations:
	//
	// * CreateLaunchTemplate
	// * ModifyLaunchTemplate
	// * DeleteLaunchTemplate
	require.NotNil(crd.Ops)

	assert.NotNil(crd.Ops.Create)
	assert.NotNil(crd.Ops.Delete)
	assert.NotNil(crd.Ops.Update)

	// However, oddly, there is no ReadOne operation. There is only a
	// ReadMany/List operation "DescribeLaunchTemplates" :(
	//
	// TODO(jaypipes): Develop strategy for informing the code generator via
	// the SDKMapper that certain APIs don't have ReadOne but only ReadMany
	// APIs...
	assert.Nil(crd.Ops.ReadOne)
	assert.NotNil(crd.Ops.ReadMany)
}

func TestECRRepository(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	sh := testutil.NewSchemaHelperForService(t, "ecr")

	crds, err := sh.GetCRDs()
	require.Nil(err)

	crd := getCRDByName("Repository", crds)
	require.NotNil(crd)

	specFields := crd.SpecFields
	statusFields := crd.StatusFields

	// The ECR API uses a REST-like API that uses "wrapper" single-member
	// objects in the JSON response for the create/describe calls. In other
	// words, the returned result from the CreateRepository API looks like
	// this:
	//
	// {
	//   "repository": {
	//	 .. bunch of fields for the repository ..
	//   }
	// }
	//
	// This test is verifying that we're properly "unwrapping" the structs and
	// putting the repository object's fields into the Spec and Status for the
	// Repository CRD.
	expSpecFieldCamel := []string{
		"ImageScanningConfiguration",
		"ImageTagMutability",
		"RepositoryName",
		"Tags",
	}
	assert.Equal(expSpecFieldCamel, attrCamelNames(specFields))

	// ImageScanningConfiguration is in the Repository resource's
	// CreateRepositoryInput shape and also returned in the
	// CreateRepositoryOutput shape, so it should produce Go code to set the
	// appropriate input shape member.
	expCreateInput := `
	if r.ko.Spec.ImageScanningConfiguration != nil {
		f0 := &svcsdk.ImageScanningConfiguration{}
		if r.ko.Spec.ImageScanningConfiguration.ScanOnPush != nil {
			f0.SetScanOnPush(*r.ko.Spec.ImageScanningConfiguration.ScanOnPush)
		}
		res.SetImageScanningConfiguration(f0)
	}
	if r.ko.Spec.ImageTagMutability != nil {
		res.SetImageTagMutability(*r.ko.Spec.ImageTagMutability)
	}
	if r.ko.Spec.RepositoryName != nil {
		res.SetRepositoryName(*r.ko.Spec.RepositoryName)
	}
	if r.ko.Spec.Tags != nil {
		f3 := []*svcsdk.Tag{}
		for _, f3iter := range r.ko.Spec.Tags {
			f3elem := &svcsdk.Tag{}
			if f3iter.Key != nil {
				f3elem.SetKey(*f3iter.Key)
			}
			if f3iter.Value != nil {
				f3elem.SetValue(*f3iter.Value)
			}
			f3 = append(f3, f3elem)
		}
		res.SetTags(f3)
	}
`
	assert.Equal(expCreateInput, crd.GoCodeSetInput(model.OpTypeCreate, "r.ko", "res", 1))

	expStatusFieldCamel := []string{
		"CreatedAt",
		// "ImageScanningConfiguration" removed because it is contained in the
		// input shape and therefore exists in the Spec
		// "ImageTagMutability" removed because it is contained in the input
		// shape and therefore exists in the Spec
		"RegistryID",
		// "RepositoryARN" removed because it refers to the primary object's
		// ARN and the SDKMapper identified it for mapping to the standard
		// Status.ACKResourceMetadata.ARN field
		// "RepositoryName" removed because it is contained in the input shape
		// and therefore exists in the Spec
		"RepositoryURI",
	}
	assert.Equal(expStatusFieldCamel, attrCamelNames(statusFields))

	// Check that we properly determined how to find the RegistryID attribute
	// within the CreateRepositoryOutput shape, which has a single field called
	// "Repository" that contains the RegistryId field.
	expCreateOutput := `
	if resp.Repository.CreatedAt != nil {
		ko.Status.CreatedAt = &metav1.Time{*resp.Repository.CreatedAt}
	}
	if resp.Repository.RegistryId != nil {
		ko.Status.RegistryID = resp.Repository.RegistryId
	}
	if resp.Repository.RepositoryUri != nil {
		ko.Status.RepositoryURI = resp.Repository.RepositoryUri
	}
`
	assert.Equal(expCreateOutput, crd.GoCodeSetOutput(model.OpTypeCreate, "resp", "ko.Status", 1))

	// The ECR Repository API has just the C and R of the normal CRUD
	// operations:
	//
	// * CreateRepository
	// * DeleteRepository
	require.NotNil(crd.Ops)

	assert.NotNil(crd.Ops.Create)
	assert.NotNil(crd.Ops.Delete)

	// There is no DescribeRepository operation. There is a List operation for
	// Repositories, though: DescribeRepositories
	assert.Nil(crd.Ops.ReadOne)
	assert.NotNil(crd.Ops.ReadMany)

	// There is no update operation (you need to call various SetXXX operations
	// on the Repository's components
	assert.Nil(crd.Ops.Update)
}

func TestCodeDeployDeployment(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	sh := testutil.NewSchemaHelperForService(t, "codedeploy")

	crds, err := sh.GetCRDs()
	require.Nil(err)

	crd := getCRDByName("Deployment", crds)
	require.NotNil(crd)

	assert.Equal("Deployment", crd.Names.Camel)
	assert.Equal("deployment", crd.Names.CamelLower)
	assert.Equal("deployment", crd.Names.Snake)

	specFields := crd.SpecFields
	statusFields := crd.StatusFields

	expSpecFieldCamel := []string{
		"ApplicationName",
		"AutoRollbackConfiguration",
		"DeploymentConfigName",
		"DeploymentGroupName",
		"Description",
		"FileExistsBehavior",
		"IgnoreApplicationStopFailures",
		"Revision",
		"TargetInstances",
		"UpdateOutdatedInstancesOnly",
	}
	assert.Equal(expSpecFieldCamel, attrCamelNames(specFields))

	// However, all of the fields in the Deployment resource's
	// CreateDeploymentInput shape are returned in the GetDeploymentOutput
	// shape, and there is a DeploymentInfo wrapper struct for the output
	// shape, so the readOne accessor contains the wrapper struct's name.
	expCreateOutput := `
	if resp.DeploymentId != nil {
		ko.Status.DeploymentID = resp.DeploymentId
	}
`
	assert.Equal(expCreateOutput, crd.GoCodeSetOutput(model.OpTypeCreate, "resp", "ko.Status", 1))

	expStatusFieldCamel := []string{
		// All of the fields in the Deployment resource's CreateDeploymentInput
		// shape are returned in the CreateDeploymentOutput shape so there are
		// not Status fields
		//
		// There is a DeploymentID field in addition to the Spec fields, though.
		"DeploymentID",
	}
	assert.Equal(expStatusFieldCamel, attrCamelNames(statusFields))

	// The CodeDeploy Deployment API actually CR+L operations:
	//
	// * CreateDeployment
	// * GetDeployment
	// * ListDeployments
	require.NotNil(crd.Ops)

	assert.NotNil(crd.Ops.Create)
	assert.NotNil(crd.Ops.ReadOne)
	assert.NotNil(crd.Ops.ReadMany)

	// But sadly, has no Update or Delete operation :(
	assert.Nil(crd.Ops.Update)
	assert.Nil(crd.Ops.Delete)
}

func TestSQSQueue(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	sh := testutil.NewSchemaHelperForService(t, "sqs")

	crds, err := sh.GetCRDs()
	require.Nil(err)

	crd := getCRDByName("Queue", crds)
	require.NotNil(crd)

	assert.Equal("Queue", crd.Names.Camel)
	assert.Equal("queue", crd.Names.CamelLower)
	assert.Equal("queue", crd.Names.Snake)

	// The SQS Queue API has CD+L operations:
	//
	// * CreateQueue
	// * DeleteQueue
	// * ListQueues
	require.NotNil(crd.Ops)

	assert.NotNil(crd.Ops.Create)
	assert.NotNil(crd.Ops.ReadMany)
	assert.NotNil(crd.Ops.Delete)
	assert.NotNil(crd.Ops.GetAttributes)
	assert.NotNil(crd.Ops.SetAttributes)

	// But sadly, has no Update or ReadOne operation :(
	// There is, however, GetQueueUrl and GetQueueAttributes calls...
	assert.Nil(crd.Ops.ReadOne)
	assert.Nil(crd.Ops.Update)

	specFields := crd.SpecFields
	statusFields := crd.StatusFields

	expSpecFieldCamel := []string{
		"ContentBasedDeduplication",
		"DelaySeconds",
		"FifoQueue",
		"KMSDataKeyReusePeriodSeconds",
		"KMSMasterKeyID",
		"MaximumMessageSize",
		"MessageRetentionPeriod",
		"Policy",
		"QueueName",
		"ReceiveMessageWaitTimeSeconds",
		"RedrivePolicy",
		"Tags",
		"VisibilityTimeout",
	}
	assert.Equal(expSpecFieldCamel, attrCamelNames(specFields))

	expStatusFieldCamel := []string{
		// There are a set of Attribute map keys that are readonly
		// fields...
		"CreatedTimestamp",
		"LastModifiedTimestamp",
		// There is only a QueueURL field returned from CreateQueueResult shape
		"QueueURL",
	}
	assert.Equal(expStatusFieldCamel, attrCamelNames(statusFields))

	expCreateInput := `
	attrMap := map[string]*string{}
	if r.ko.Spec.ContentBasedDeduplication != nil {
		attrMap["ContentBasedDeduplication"] = r.ko.Spec.ContentBasedDeduplication
	}
	if r.ko.Spec.DelaySeconds != nil {
		attrMap["DelaySeconds"] = r.ko.Spec.DelaySeconds
	}
	if r.ko.Spec.FifoQueue != nil {
		attrMap["FifoQueue"] = r.ko.Spec.FifoQueue
	}
	if r.ko.Spec.KMSDataKeyReusePeriodSeconds != nil {
		attrMap["KmsDataKeyReusePeriodSeconds"] = r.ko.Spec.KMSDataKeyReusePeriodSeconds
	}
	if r.ko.Spec.KMSMasterKeyID != nil {
		attrMap["KmsMasterKeyId"] = r.ko.Spec.KMSMasterKeyID
	}
	if r.ko.Spec.MaximumMessageSize != nil {
		attrMap["MaximumMessageSize"] = r.ko.Spec.MaximumMessageSize
	}
	if r.ko.Spec.MessageRetentionPeriod != nil {
		attrMap["MessageRetentionPeriod"] = r.ko.Spec.MessageRetentionPeriod
	}
	if r.ko.Spec.Policy != nil {
		attrMap["Policy"] = r.ko.Spec.Policy
	}
	if r.ko.Spec.ReceiveMessageWaitTimeSeconds != nil {
		attrMap["ReceiveMessageWaitTimeSeconds"] = r.ko.Spec.ReceiveMessageWaitTimeSeconds
	}
	if r.ko.Spec.RedrivePolicy != nil {
		attrMap["RedrivePolicy"] = r.ko.Spec.RedrivePolicy
	}
	if r.ko.Spec.VisibilityTimeout != nil {
		attrMap["VisibilityTimeout"] = r.ko.Spec.VisibilityTimeout
	}
	res.SetAttributes(attrMap)
	if r.ko.Spec.QueueName != nil {
		res.SetQueueName(*r.ko.Spec.QueueName)
	}
	if r.ko.Spec.Tags != nil {
		f2 := map[string]*string{}
		for f2key, f2valiter := range r.ko.Spec.Tags {
			var f2val string
			f2val = *f2valiter
			f2[f2key] = &f2val
		}
		res.SetTags(f2)
	}
`
	assert.Equal(expCreateInput, crd.GoCodeSetInput(model.OpTypeCreate, "r.ko", "res", 1))

	// There are no fields other than QueueID in the returned CreateQueueResult
	// shape
	expCreateOutput := `
	if resp.QueueUrl != nil {
		ko.Status.QueueURL = resp.QueueUrl
	}
`
	assert.Equal(expCreateOutput, crd.GoCodeSetOutput(model.OpTypeCreate, "resp", "ko.Status", 1))

	// The input shape for the GetAttributes operation technically has two
	// fields in it: an AttributeNames list of attribute keys to file
	// attributes for and a QueueUrl field. We only care about the QueueUrl
	// field, since we look for all attributes for a queue.
	expGetAttrsInput := `
	if r.ko.Status.QueueURL != nil {
		res.SetQueueUrl(*r.ko.Status.QueueURL)
	}
`
	assert.Equal(expGetAttrsInput, crd.GoCodeGetAttributesSetInput("r.ko", "res", 1))

	// The output shape for the GetAttributes operation contains a single field
	// "Attributes" that must be unpacked into the Queue CRD's Status fields.
	// There are only three attribute keys that are *not* in the Input shape
	// (and thus in the Spec fields). One of them is the resource's ARN which
	// is handled specially.
	expGetAttrsOutput := `
	ko.Status.CreatedTimestamp = resp.Attributes["CreatedTimestamp"]
	ko.Status.LastModifiedTimestamp = resp.Attributes["LastModifiedTimestamp"]
	tmpARN := ackv1alpha1.AWSResourceName(*resp.Attributes["QueueArn"])
	ko.Status.ACKResourceMetadata.ARN = &tmpARN
`
	assert.Equal(expGetAttrsOutput, crd.GoCodeGetAttributesSetOutput("resp", "ko.Status", 1))
}

func TestAPIGatewayV2_Route(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	sh := testutil.NewSchemaHelperForService(t, "apigatewayv2")

	crds, err := sh.GetCRDs()
	require.Nil(err)

	crd := getCRDByName("Route", crds)
	require.NotNil(crd)

	assert.Equal("Route", crd.Names.Camel)
	assert.Equal("route", crd.Names.CamelLower)
	assert.Equal("route", crd.Names.Snake)

	// The APIGatewayV2 Route API has CRUD+L operations:
	//
	// * CreateRoute
	// * DeleteRoute
	// * UpdateRoute
	// * GetRoute
	// * GetRoutes
	require.NotNil(crd.Ops)

	assert.NotNil(crd.Ops.Create)
	assert.NotNil(crd.Ops.Delete)
	assert.NotNil(crd.Ops.Update)
	assert.NotNil(crd.Ops.ReadOne)
	assert.NotNil(crd.Ops.ReadMany)

	// And no separate get/set attributes calls.
	assert.Nil(crd.Ops.GetAttributes)
	assert.Nil(crd.Ops.SetAttributes)

	specFields := crd.SpecFields
	statusFields := crd.StatusFields

	expSpecFieldCamel := []string{
		"APIID",
		"APIKeyRequired",
		"AuthorizationScopes",
		"AuthorizationType",
		"AuthorizerID",
		"ModelSelectionExpression",
		"OperationName",
		"RequestModels",
		"RequestParameters",
		"RouteKey",
		"RouteResponseSelectionExpression",
		"Target",
	}
	assert.Equal(expSpecFieldCamel, attrCamelNames(specFields))

	expStatusFieldCamel := []string{
		"APIGatewayManaged",
		"RouteID",
	}
	assert.Equal(expStatusFieldCamel, attrCamelNames(statusFields))

	expCreateInput := `
	if r.ko.Spec.APIID != nil {
		res.SetApiId(*r.ko.Spec.APIID)
	}
	if r.ko.Spec.APIKeyRequired != nil {
		res.SetApiKeyRequired(*r.ko.Spec.APIKeyRequired)
	}
	if r.ko.Spec.AuthorizationScopes != nil {
		f2 := []*string{}
		for _, f2iter := range r.ko.Spec.AuthorizationScopes {
			var f2elem string
			f2elem = *f2iter
			f2 = append(f2, &f2elem)
		}
		res.SetAuthorizationScopes(f2)
	}
	if r.ko.Spec.AuthorizationType != nil {
		res.SetAuthorizationType(*r.ko.Spec.AuthorizationType)
	}
	if r.ko.Spec.AuthorizerID != nil {
		res.SetAuthorizerId(*r.ko.Spec.AuthorizerID)
	}
	if r.ko.Spec.ModelSelectionExpression != nil {
		res.SetModelSelectionExpression(*r.ko.Spec.ModelSelectionExpression)
	}
	if r.ko.Spec.OperationName != nil {
		res.SetOperationName(*r.ko.Spec.OperationName)
	}
	if r.ko.Spec.RequestModels != nil {
		f7 := map[string]*string{}
		for f7key, f7valiter := range r.ko.Spec.RequestModels {
			var f7val string
			f7val = *f7valiter
			f7[f7key] = &f7val
		}
		res.SetRequestModels(f7)
	}
	if r.ko.Spec.RequestParameters != nil {
		f8 := map[string]*svcsdk.ParameterConstraints{}
		for f8key, f8valiter := range r.ko.Spec.RequestParameters {
			f8val := &svcsdk.ParameterConstraints{}
			if f8valiter.Required != nil {
				f8val.SetRequired(*f8valiter.Required)
			}
			f8[f8key] = f8val
		}
		res.SetRequestParameters(f8)
	}
	if r.ko.Spec.RouteKey != nil {
		res.SetRouteKey(*r.ko.Spec.RouteKey)
	}
	if r.ko.Spec.RouteResponseSelectionExpression != nil {
		res.SetRouteResponseSelectionExpression(*r.ko.Spec.RouteResponseSelectionExpression)
	}
	if r.ko.Spec.Target != nil {
		res.SetTarget(*r.ko.Spec.Target)
	}
`
	assert.Equal(expCreateInput, crd.GoCodeSetInput(model.OpTypeCreate, "r.ko", "res", 1))

	expCreateOutput := `
	if resp.ApiGatewayManaged != nil {
		ko.Status.APIGatewayManaged = resp.ApiGatewayManaged
	}
	if resp.RouteId != nil {
		ko.Status.RouteID = resp.RouteId
	}
`
	assert.Equal(expCreateOutput, crd.GoCodeSetOutput(model.OpTypeCreate, "resp", "ko.Status", 1))
}

func TestElasticache_CacheCluster(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	sh := testutil.NewSchemaHelperForService(t, "elasticache")

	crds, err := sh.GetCRDs()
	require.Nil(err)

	crd := getCRDByName("CacheCluster", crds)
	require.NotNil(crd)

	assert.Equal("CacheCluster", crd.Names.Camel)
	assert.Equal("cacheCluster", crd.Names.CamelLower)
	assert.Equal("cache_cluster", crd.Names.Snake)

	// The Elasticache CacheCluster API has CUD+L operations:
	//
	// * CreateCacheCluster
	// * DeleteCacheCluster
	// * UpdateCacheCluster
	// * GetCacheClusters
	require.NotNil(crd.Ops)

	assert.NotNil(crd.Ops.Create)
	assert.NotNil(crd.Ops.Delete)
	assert.NotNil(crd.Ops.Update)
	assert.NotNil(crd.Ops.ReadMany)

	// But no ReadOne operation...
	assert.Nil(crd.Ops.ReadOne)

	// And no separate get/set attributes calls.
	assert.Nil(crd.Ops.GetAttributes)
	assert.Nil(crd.Ops.SetAttributes)

	specFields := crd.SpecFields
	statusFields := crd.StatusFields

	expSpecFieldCamel := []string{
		"AZMode",
		"AuthToken",
		"AutoMinorVersionUpgrade",
		"CacheClusterID",
		"CacheNodeType",
		"CacheParameterGroupName",
		"CacheSecurityGroupNames",
		"CacheSubnetGroupName",
		"Engine",
		"EngineVersion",
		"NotificationTopicARN",
		"NumCacheNodes",
		"Port",
		"PreferredAvailabilityZone",
		"PreferredAvailabilityZones",
		"PreferredMaintenanceWindow",
		"ReplicationGroupID",
		"SecurityGroupIDs",
		"SnapshotARNs",
		"SnapshotName",
		"SnapshotRetentionLimit",
		"SnapshotWindow",
		"Tags",
	}
	assert.Equal(expSpecFieldCamel, attrCamelNames(specFields))

	expStatusFieldCamel := []string{
		"AtRestEncryptionEnabled",
		"AuthTokenEnabled",
		"AuthTokenLastModifiedDate",
		"CacheClusterCreateTime",
		"CacheClusterStatus",
		"CacheNodes",
		"CacheParameterGroup",
		"CacheSecurityGroups",
		"ClientDownloadLandingPage",
		"ConfigurationEndpoint",
		"NotificationConfiguration",
		"PendingModifiedValues",
		"SecurityGroups",
		"TransitEncryptionEnabled",
	}
	assert.Equal(expStatusFieldCamel, attrCamelNames(statusFields))

	expCreateInput := `
	if r.ko.Spec.AZMode != nil {
		res.SetAZMode(*r.ko.Spec.AZMode)
	}
	if r.ko.Spec.AuthToken != nil {
		res.SetAuthToken(*r.ko.Spec.AuthToken)
	}
	if r.ko.Spec.AutoMinorVersionUpgrade != nil {
		res.SetAutoMinorVersionUpgrade(*r.ko.Spec.AutoMinorVersionUpgrade)
	}
	if r.ko.Spec.CacheClusterID != nil {
		res.SetCacheClusterId(*r.ko.Spec.CacheClusterID)
	}
	if r.ko.Spec.CacheNodeType != nil {
		res.SetCacheNodeType(*r.ko.Spec.CacheNodeType)
	}
	if r.ko.Spec.CacheParameterGroupName != nil {
		res.SetCacheParameterGroupName(*r.ko.Spec.CacheParameterGroupName)
	}
	if r.ko.Spec.CacheSecurityGroupNames != nil {
		f6 := []*string{}
		for _, f6iter := range r.ko.Spec.CacheSecurityGroupNames {
			var f6elem string
			f6elem = *f6iter
			f6 = append(f6, &f6elem)
		}
		res.SetCacheSecurityGroupNames(f6)
	}
	if r.ko.Spec.CacheSubnetGroupName != nil {
		res.SetCacheSubnetGroupName(*r.ko.Spec.CacheSubnetGroupName)
	}
	if r.ko.Spec.Engine != nil {
		res.SetEngine(*r.ko.Spec.Engine)
	}
	if r.ko.Spec.EngineVersion != nil {
		res.SetEngineVersion(*r.ko.Spec.EngineVersion)
	}
	if r.ko.Spec.NotificationTopicARN != nil {
		res.SetNotificationTopicArn(*r.ko.Spec.NotificationTopicARN)
	}
	if r.ko.Spec.NumCacheNodes != nil {
		res.SetNumCacheNodes(*r.ko.Spec.NumCacheNodes)
	}
	if r.ko.Spec.Port != nil {
		res.SetPort(*r.ko.Spec.Port)
	}
	if r.ko.Spec.PreferredAvailabilityZone != nil {
		res.SetPreferredAvailabilityZone(*r.ko.Spec.PreferredAvailabilityZone)
	}
	if r.ko.Spec.PreferredAvailabilityZones != nil {
		f14 := []*string{}
		for _, f14iter := range r.ko.Spec.PreferredAvailabilityZones {
			var f14elem string
			f14elem = *f14iter
			f14 = append(f14, &f14elem)
		}
		res.SetPreferredAvailabilityZones(f14)
	}
	if r.ko.Spec.PreferredMaintenanceWindow != nil {
		res.SetPreferredMaintenanceWindow(*r.ko.Spec.PreferredMaintenanceWindow)
	}
	if r.ko.Spec.ReplicationGroupID != nil {
		res.SetReplicationGroupId(*r.ko.Spec.ReplicationGroupID)
	}
	if r.ko.Spec.SecurityGroupIDs != nil {
		f17 := []*string{}
		for _, f17iter := range r.ko.Spec.SecurityGroupIDs {
			var f17elem string
			f17elem = *f17iter
			f17 = append(f17, &f17elem)
		}
		res.SetSecurityGroupIds(f17)
	}
	if r.ko.Spec.SnapshotARNs != nil {
		f18 := []*string{}
		for _, f18iter := range r.ko.Spec.SnapshotARNs {
			var f18elem string
			f18elem = *f18iter
			f18 = append(f18, &f18elem)
		}
		res.SetSnapshotArns(f18)
	}
	if r.ko.Spec.SnapshotName != nil {
		res.SetSnapshotName(*r.ko.Spec.SnapshotName)
	}
	if r.ko.Spec.SnapshotRetentionLimit != nil {
		res.SetSnapshotRetentionLimit(*r.ko.Spec.SnapshotRetentionLimit)
	}
	if r.ko.Spec.SnapshotWindow != nil {
		res.SetSnapshotWindow(*r.ko.Spec.SnapshotWindow)
	}
	if r.ko.Spec.Tags != nil {
		f22 := []*svcsdk.Tag{}
		for _, f22iter := range r.ko.Spec.Tags {
			f22elem := &svcsdk.Tag{}
			if f22iter.Key != nil {
				f22elem.SetKey(*f22iter.Key)
			}
			if f22iter.Value != nil {
				f22elem.SetValue(*f22iter.Value)
			}
			f22 = append(f22, f22elem)
		}
		res.SetTags(f22)
	}
`
	assert.Equal(expCreateInput, crd.GoCodeSetInput(model.OpTypeCreate, "r.ko", "res", 1))

	expCreateOutput := `
	if resp.CacheCluster.AtRestEncryptionEnabled != nil {
		ko.Status.AtRestEncryptionEnabled = resp.CacheCluster.AtRestEncryptionEnabled
	}
	if resp.CacheCluster.AuthTokenEnabled != nil {
		ko.Status.AuthTokenEnabled = resp.CacheCluster.AuthTokenEnabled
	}
	if resp.CacheCluster.AuthTokenLastModifiedDate != nil {
		ko.Status.AuthTokenLastModifiedDate = &metav1.Time{*resp.CacheCluster.AuthTokenLastModifiedDate}
	}
	if resp.CacheCluster.CacheClusterCreateTime != nil {
		ko.Status.CacheClusterCreateTime = &metav1.Time{*resp.CacheCluster.CacheClusterCreateTime}
	}
	if resp.CacheCluster.CacheClusterStatus != nil {
		ko.Status.CacheClusterStatus = resp.CacheCluster.CacheClusterStatus
	}
	if resp.CacheCluster.CacheNodes != nil {
		f9 := []*svcapitypes.CacheNode{}
		for _, f9iter := range resp.CacheCluster.CacheNodes {
			f9elem := &svcapitypes.CacheNode{}
			if f9iter.CacheNodeCreateTime != nil {
				f9elem.CacheNodeCreateTime = &metav1.Time{*f9iter.CacheNodeCreateTime}
			}
			if f9iter.CacheNodeId != nil {
				f9elem.CacheNodeID = f9iter.CacheNodeId
			}
			if f9iter.CacheNodeStatus != nil {
				f9elem.CacheNodeStatus = f9iter.CacheNodeStatus
			}
			if f9iter.CustomerAvailabilityZone != nil {
				f9elem.CustomerAvailabilityZone = f9iter.CustomerAvailabilityZone
			}
			if f9iter.Endpoint != nil {
				f9elemf4 := &svcapitypes.Endpoint{}
				if f9iter.Endpoint.Address != nil {
					f9elemf4.Address = f9iter.Endpoint.Address
				}
				if f9iter.Endpoint.Port != nil {
					f9elemf4.Port = f9iter.Endpoint.Port
				}
				f9elem.Endpoint = f9elemf4
			}
			if f9iter.ParameterGroupStatus != nil {
				f9elem.ParameterGroupStatus = f9iter.ParameterGroupStatus
			}
			if f9iter.SourceCacheNodeId != nil {
				f9elem.SourceCacheNodeID = f9iter.SourceCacheNodeId
			}
			f9 = append(f9, f9elem)
		}
		ko.Status.CacheNodes = f9
	}
	if resp.CacheCluster.CacheParameterGroup != nil {
		f10 := &svcapitypes.CacheParameterGroupStatus_SDK{}
		if resp.CacheCluster.CacheParameterGroup.CacheNodeIdsToReboot != nil {
			f10f0 := []*string{}
			for _, f10f0iter := range resp.CacheCluster.CacheParameterGroup.CacheNodeIdsToReboot {
				var f10f0elem string
				f10f0elem = *f10f0iter
				f10f0 = append(f10f0, &f10f0elem)
			}
			f10.CacheNodeIDsToReboot = f10f0
		}
		if resp.CacheCluster.CacheParameterGroup.CacheParameterGroupName != nil {
			f10.CacheParameterGroupName = resp.CacheCluster.CacheParameterGroup.CacheParameterGroupName
		}
		if resp.CacheCluster.CacheParameterGroup.ParameterApplyStatus != nil {
			f10.ParameterApplyStatus = resp.CacheCluster.CacheParameterGroup.ParameterApplyStatus
		}
		ko.Status.CacheParameterGroup = f10
	}
	if resp.CacheCluster.CacheSecurityGroups != nil {
		f11 := []*svcapitypes.CacheSecurityGroupMembership{}
		for _, f11iter := range resp.CacheCluster.CacheSecurityGroups {
			f11elem := &svcapitypes.CacheSecurityGroupMembership{}
			if f11iter.CacheSecurityGroupName != nil {
				f11elem.CacheSecurityGroupName = f11iter.CacheSecurityGroupName
			}
			if f11iter.Status != nil {
				f11elem.Status = f11iter.Status
			}
			f11 = append(f11, f11elem)
		}
		ko.Status.CacheSecurityGroups = f11
	}
	if resp.CacheCluster.ClientDownloadLandingPage != nil {
		ko.Status.ClientDownloadLandingPage = resp.CacheCluster.ClientDownloadLandingPage
	}
	if resp.CacheCluster.ConfigurationEndpoint != nil {
		f14 := &svcapitypes.Endpoint{}
		if resp.CacheCluster.ConfigurationEndpoint.Address != nil {
			f14.Address = resp.CacheCluster.ConfigurationEndpoint.Address
		}
		if resp.CacheCluster.ConfigurationEndpoint.Port != nil {
			f14.Port = resp.CacheCluster.ConfigurationEndpoint.Port
		}
		ko.Status.ConfigurationEndpoint = f14
	}
	if resp.CacheCluster.NotificationConfiguration != nil {
		f17 := &svcapitypes.NotificationConfiguration{}
		if resp.CacheCluster.NotificationConfiguration.TopicArn != nil {
			f17.TopicARN = resp.CacheCluster.NotificationConfiguration.TopicArn
		}
		if resp.CacheCluster.NotificationConfiguration.TopicStatus != nil {
			f17.TopicStatus = resp.CacheCluster.NotificationConfiguration.TopicStatus
		}
		ko.Status.NotificationConfiguration = f17
	}
	if resp.CacheCluster.PendingModifiedValues != nil {
		f19 := &svcapitypes.PendingModifiedValues{}
		if resp.CacheCluster.PendingModifiedValues.AuthTokenStatus != nil {
			f19.AuthTokenStatus = resp.CacheCluster.PendingModifiedValues.AuthTokenStatus
		}
		if resp.CacheCluster.PendingModifiedValues.CacheNodeIdsToRemove != nil {
			f19f1 := []*string{}
			for _, f19f1iter := range resp.CacheCluster.PendingModifiedValues.CacheNodeIdsToRemove {
				var f19f1elem string
				f19f1elem = *f19f1iter
				f19f1 = append(f19f1, &f19f1elem)
			}
			f19.CacheNodeIDsToRemove = f19f1
		}
		if resp.CacheCluster.PendingModifiedValues.CacheNodeType != nil {
			f19.CacheNodeType = resp.CacheCluster.PendingModifiedValues.CacheNodeType
		}
		if resp.CacheCluster.PendingModifiedValues.EngineVersion != nil {
			f19.EngineVersion = resp.CacheCluster.PendingModifiedValues.EngineVersion
		}
		if resp.CacheCluster.PendingModifiedValues.NumCacheNodes != nil {
			f19.NumCacheNodes = resp.CacheCluster.PendingModifiedValues.NumCacheNodes
		}
		ko.Status.PendingModifiedValues = f19
	}
	if resp.CacheCluster.SecurityGroups != nil {
		f23 := []*svcapitypes.SecurityGroupMembership{}
		for _, f23iter := range resp.CacheCluster.SecurityGroups {
			f23elem := &svcapitypes.SecurityGroupMembership{}
			if f23iter.SecurityGroupId != nil {
				f23elem.SecurityGroupID = f23iter.SecurityGroupId
			}
			if f23iter.Status != nil {
				f23elem.Status = f23iter.Status
			}
			f23 = append(f23, f23elem)
		}
		ko.Status.SecurityGroups = f23
	}
	if resp.CacheCluster.TransitEncryptionEnabled != nil {
		ko.Status.TransitEncryptionEnabled = resp.CacheCluster.TransitEncryptionEnabled
	}
`
	assert.Equal(expCreateOutput, crd.GoCodeSetOutput(model.OpTypeCreate, "resp", "ko.Status", 1))

	// Elasticache doesn't have a ReadOne operation; only a List/ReadMany
	// operation. Let's verify that the construction of the
	// DescribeCacheClustersInput and processing of the
	// DescribeCacheClustersOutput shapes is correct.
	expReadManyInput := `
	if r.ko.Spec.CacheClusterID != nil {
		res.SetCacheClusterId(*r.ko.Spec.CacheClusterID)
	}
`
	assert.Equal(expReadManyInput, crd.GoCodeSetInput(model.OpTypeList, "r.ko", "res", 1))

	expReadManyOutput := `
	if len(resp.CacheClusters) == 1 {
		elem := resp.CacheClusters[0]
		if elem.ARN != nil {
			if ko.Status.ACKResourceMetadata == nil {
				ko.Status.ACKResourceMetadata = &ackv1alpha1.ResourceMetadata{}
			}
			tmpARN := ackv1alpha1.AWSResourceName(*elem.ARN)
			ko.Status.ACKResourceMetadata.ARN = &tmpARN
		}
		if elem.AtRestEncryptionEnabled != nil {
			ko.Status.AtRestEncryptionEnabled = elem.AtRestEncryptionEnabled
		}
		if elem.AuthTokenEnabled != nil {
			ko.Status.AuthTokenEnabled = elem.AuthTokenEnabled
		}
		if elem.AuthTokenLastModifiedDate != nil {
			ko.Status.AuthTokenLastModifiedDate = &metav1.Time{*elem.AuthTokenLastModifiedDate}
		}
		if elem.AutoMinorVersionUpgrade != nil {
			ko.Spec.AutoMinorVersionUpgrade = elem.AutoMinorVersionUpgrade
		}
		if elem.CacheClusterCreateTime != nil {
			ko.Status.CacheClusterCreateTime = &metav1.Time{*elem.CacheClusterCreateTime}
		}
		if elem.CacheClusterId != nil {
			ko.Spec.CacheClusterID = elem.CacheClusterId
		}
		if elem.CacheClusterStatus != nil {
			ko.Status.CacheClusterStatus = elem.CacheClusterStatus
		}
		if elem.CacheNodeType != nil {
			ko.Spec.CacheNodeType = elem.CacheNodeType
		}
		if elem.CacheNodes != nil {
			f9 := []*svcapitypes.CacheNode{}
			for _, f9iter := range elem.CacheNodes {
				f9elem := &svcapitypes.CacheNode{}
				if f9iter.CacheNodeCreateTime != nil {
					f9elem.CacheNodeCreateTime = &metav1.Time{*f9iter.CacheNodeCreateTime}
				}
				if f9iter.CacheNodeId != nil {
					f9elem.CacheNodeID = f9iter.CacheNodeId
				}
				if f9iter.CacheNodeStatus != nil {
					f9elem.CacheNodeStatus = f9iter.CacheNodeStatus
				}
				if f9iter.CustomerAvailabilityZone != nil {
					f9elem.CustomerAvailabilityZone = f9iter.CustomerAvailabilityZone
				}
				if f9iter.Endpoint != nil {
					f9elemf4 := &svcapitypes.Endpoint{}
					if f9iter.Endpoint.Address != nil {
						f9elemf4.Address = f9iter.Endpoint.Address
					}
					if f9iter.Endpoint.Port != nil {
						f9elemf4.Port = f9iter.Endpoint.Port
					}
					f9elem.Endpoint = f9elemf4
				}
				if f9iter.ParameterGroupStatus != nil {
					f9elem.ParameterGroupStatus = f9iter.ParameterGroupStatus
				}
				if f9iter.SourceCacheNodeId != nil {
					f9elem.SourceCacheNodeID = f9iter.SourceCacheNodeId
				}
				f9 = append(f9, f9elem)
			}
			ko.Status.CacheNodes = f9
		}
		if elem.CacheParameterGroup != nil {
			f10 := &svcapitypes.CacheParameterGroupStatus_SDK{}
			if elem.CacheParameterGroup.CacheNodeIdsToReboot != nil {
				f10f0 := []*string{}
				for _, f10f0iter := range elem.CacheParameterGroup.CacheNodeIdsToReboot {
					var f10f0elem string
					f10f0elem = *f10f0iter
					f10f0 = append(f10f0, &f10f0elem)
				}
				f10.CacheNodeIDsToReboot = f10f0
			}
			if elem.CacheParameterGroup.CacheParameterGroupName != nil {
				f10.CacheParameterGroupName = elem.CacheParameterGroup.CacheParameterGroupName
			}
			if elem.CacheParameterGroup.ParameterApplyStatus != nil {
				f10.ParameterApplyStatus = elem.CacheParameterGroup.ParameterApplyStatus
			}
			ko.Status.CacheParameterGroup = f10
		}
		if elem.CacheSecurityGroups != nil {
			f11 := []*svcapitypes.CacheSecurityGroupMembership{}
			for _, f11iter := range elem.CacheSecurityGroups {
				f11elem := &svcapitypes.CacheSecurityGroupMembership{}
				if f11iter.CacheSecurityGroupName != nil {
					f11elem.CacheSecurityGroupName = f11iter.CacheSecurityGroupName
				}
				if f11iter.Status != nil {
					f11elem.Status = f11iter.Status
				}
				f11 = append(f11, f11elem)
			}
			ko.Status.CacheSecurityGroups = f11
		}
		if elem.CacheSubnetGroupName != nil {
			ko.Spec.CacheSubnetGroupName = elem.CacheSubnetGroupName
		}
		if elem.ClientDownloadLandingPage != nil {
			ko.Status.ClientDownloadLandingPage = elem.ClientDownloadLandingPage
		}
		if elem.ConfigurationEndpoint != nil {
			f14 := &svcapitypes.Endpoint{}
			if elem.ConfigurationEndpoint.Address != nil {
				f14.Address = elem.ConfigurationEndpoint.Address
			}
			if elem.ConfigurationEndpoint.Port != nil {
				f14.Port = elem.ConfigurationEndpoint.Port
			}
			ko.Status.ConfigurationEndpoint = f14
		}
		if elem.Engine != nil {
			ko.Spec.Engine = elem.Engine
		}
		if elem.EngineVersion != nil {
			ko.Spec.EngineVersion = elem.EngineVersion
		}
		if elem.NotificationConfiguration != nil {
			f17 := &svcapitypes.NotificationConfiguration{}
			if elem.NotificationConfiguration.TopicArn != nil {
				f17.TopicARN = elem.NotificationConfiguration.TopicArn
			}
			if elem.NotificationConfiguration.TopicStatus != nil {
				f17.TopicStatus = elem.NotificationConfiguration.TopicStatus
			}
			ko.Status.NotificationConfiguration = f17
		}
		if elem.NumCacheNodes != nil {
			ko.Spec.NumCacheNodes = elem.NumCacheNodes
		}
		if elem.PendingModifiedValues != nil {
			f19 := &svcapitypes.PendingModifiedValues{}
			if elem.PendingModifiedValues.AuthTokenStatus != nil {
				f19.AuthTokenStatus = elem.PendingModifiedValues.AuthTokenStatus
			}
			if elem.PendingModifiedValues.CacheNodeIdsToRemove != nil {
				f19f1 := []*string{}
				for _, f19f1iter := range elem.PendingModifiedValues.CacheNodeIdsToRemove {
					var f19f1elem string
					f19f1elem = *f19f1iter
					f19f1 = append(f19f1, &f19f1elem)
				}
				f19.CacheNodeIDsToRemove = f19f1
			}
			if elem.PendingModifiedValues.CacheNodeType != nil {
				f19.CacheNodeType = elem.PendingModifiedValues.CacheNodeType
			}
			if elem.PendingModifiedValues.EngineVersion != nil {
				f19.EngineVersion = elem.PendingModifiedValues.EngineVersion
			}
			if elem.PendingModifiedValues.NumCacheNodes != nil {
				f19.NumCacheNodes = elem.PendingModifiedValues.NumCacheNodes
			}
			ko.Status.PendingModifiedValues = f19
		}
		if elem.PreferredAvailabilityZone != nil {
			ko.Spec.PreferredAvailabilityZone = elem.PreferredAvailabilityZone
		}
		if elem.PreferredMaintenanceWindow != nil {
			ko.Spec.PreferredMaintenanceWindow = elem.PreferredMaintenanceWindow
		}
		if elem.ReplicationGroupId != nil {
			ko.Spec.ReplicationGroupID = elem.ReplicationGroupId
		}
		if elem.SecurityGroups != nil {
			f23 := []*svcapitypes.SecurityGroupMembership{}
			for _, f23iter := range elem.SecurityGroups {
				f23elem := &svcapitypes.SecurityGroupMembership{}
				if f23iter.SecurityGroupId != nil {
					f23elem.SecurityGroupID = f23iter.SecurityGroupId
				}
				if f23iter.Status != nil {
					f23elem.Status = f23iter.Status
				}
				f23 = append(f23, f23elem)
			}
			ko.Status.SecurityGroups = f23
		}
		if elem.SnapshotRetentionLimit != nil {
			ko.Spec.SnapshotRetentionLimit = elem.SnapshotRetentionLimit
		}
		if elem.SnapshotWindow != nil {
			ko.Spec.SnapshotWindow = elem.SnapshotWindow
		}
		if elem.TransitEncryptionEnabled != nil {
			ko.Status.TransitEncryptionEnabled = elem.TransitEncryptionEnabled
		}
	}
`
	assert.Equal(expReadManyOutput, crd.GoCodeSetOutput(model.OpTypeList, "resp", "ko", 1))
}

func TestElasticache_Ignored_Operations(t *testing.T) {
	require := require.New(t)

	sh := testutil.NewSchemaHelperForService(t, "elasticache")

	crds, err := sh.GetCRDs()
	require.Nil(err)

	crd := getCRDByName("Snapshot", crds)
	require.NotNil(crd)
	require.NotNil(crd.Ops.Create)
	require.Nil(crd.Ops.Delete)
}

func TestElasticache_Ignored_Resources(t *testing.T) {
	require := require.New(t)

	sh := testutil.NewSchemaHelperForService(t, "elasticache")

	crds, err := sh.GetCRDs()
	require.Nil(err)

	crd := getCRDByName("GlobalReplicationGroup", crds)
	require.Nil(crd)
}

func TestDynamoDB_Table(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	sh := testutil.NewSchemaHelperForService(t, "dynamodb")

	crds, err := sh.GetCRDs()
	require.Nil(err)

	crd := getCRDByName("Table", crds)
	require.NotNil(crd)

	// The DynamoDB Table API has these operations:
	//
	// * CreateTable
	// * DeleteTable
	// * DescribeTable
	// * ListTables
	// * UpdateTable
	require.NotNil(crd.Ops)

	assert.NotNil(crd.Ops.Create)
	assert.NotNil(crd.Ops.Delete)
	assert.NotNil(crd.Ops.ReadOne)
	assert.NotNil(crd.Ops.ReadMany)
	assert.NotNil(crd.Ops.Update)

	assert.Nil(crd.Ops.GetAttributes)
	assert.Nil(crd.Ops.SetAttributes)

	specFields := crd.SpecFields
	statusFields := crd.StatusFields

	expSpecFieldCamel := []string{
		"AttributeDefinitions",
		"BillingMode",
		"GlobalSecondaryIndexes",
		"KeySchema",
		"LocalSecondaryIndexes",
		"ProvisionedThroughput",
		"SSESpecification",
		"StreamSpecification",
		"TableName",
		"Tags",
	}
	assert.Equal(expSpecFieldCamel, attrCamelNames(specFields))

	expCreateInput := `
	if r.ko.Spec.AttributeDefinitions != nil {
		f0 := []*svcsdk.AttributeDefinition{}
		for _, f0iter := range r.ko.Spec.AttributeDefinitions {
			f0elem := &svcsdk.AttributeDefinition{}
			if f0iter.AttributeName != nil {
				f0elem.SetAttributeName(*f0iter.AttributeName)
			}
			if f0iter.AttributeType != nil {
				f0elem.SetAttributeType(*f0iter.AttributeType)
			}
			f0 = append(f0, f0elem)
		}
		res.SetAttributeDefinitions(f0)
	}
	if r.ko.Spec.BillingMode != nil {
		res.SetBillingMode(*r.ko.Spec.BillingMode)
	}
	if r.ko.Spec.GlobalSecondaryIndexes != nil {
		f2 := []*svcsdk.GlobalSecondaryIndex{}
		for _, f2iter := range r.ko.Spec.GlobalSecondaryIndexes {
			f2elem := &svcsdk.GlobalSecondaryIndex{}
			if f2iter.IndexName != nil {
				f2elem.SetIndexName(*f2iter.IndexName)
			}
			if f2iter.KeySchema != nil {
				f2elemf1 := []*svcsdk.KeySchemaElement{}
				for _, f2elemf1iter := range f2iter.KeySchema {
					f2elemf1elem := &svcsdk.KeySchemaElement{}
					if f2elemf1iter.AttributeName != nil {
						f2elemf1elem.SetAttributeName(*f2elemf1iter.AttributeName)
					}
					if f2elemf1iter.KeyType != nil {
						f2elemf1elem.SetKeyType(*f2elemf1iter.KeyType)
					}
					f2elemf1 = append(f2elemf1, f2elemf1elem)
				}
				f2elem.SetKeySchema(f2elemf1)
			}
			if f2iter.Projection != nil {
				f2elemf2 := &svcsdk.Projection{}
				if f2iter.Projection.NonKeyAttributes != nil {
					f2elemf2f0 := []*string{}
					for _, f2elemf2f0iter := range f2iter.Projection.NonKeyAttributes {
						var f2elemf2f0elem string
						f2elemf2f0elem = *f2elemf2f0iter
						f2elemf2f0 = append(f2elemf2f0, &f2elemf2f0elem)
					}
					f2elemf2.SetNonKeyAttributes(f2elemf2f0)
				}
				if f2iter.Projection.ProjectionType != nil {
					f2elemf2.SetProjectionType(*f2iter.Projection.ProjectionType)
				}
				f2elem.SetProjection(f2elemf2)
			}
			if f2iter.ProvisionedThroughput != nil {
				f2elemf3 := &svcsdk.ProvisionedThroughput{}
				if f2iter.ProvisionedThroughput.ReadCapacityUnits != nil {
					f2elemf3.SetReadCapacityUnits(*f2iter.ProvisionedThroughput.ReadCapacityUnits)
				}
				if f2iter.ProvisionedThroughput.WriteCapacityUnits != nil {
					f2elemf3.SetWriteCapacityUnits(*f2iter.ProvisionedThroughput.WriteCapacityUnits)
				}
				f2elem.SetProvisionedThroughput(f2elemf3)
			}
			f2 = append(f2, f2elem)
		}
		res.SetGlobalSecondaryIndexes(f2)
	}
	if r.ko.Spec.KeySchema != nil {
		f3 := []*svcsdk.KeySchemaElement{}
		for _, f3iter := range r.ko.Spec.KeySchema {
			f3elem := &svcsdk.KeySchemaElement{}
			if f3iter.AttributeName != nil {
				f3elem.SetAttributeName(*f3iter.AttributeName)
			}
			if f3iter.KeyType != nil {
				f3elem.SetKeyType(*f3iter.KeyType)
			}
			f3 = append(f3, f3elem)
		}
		res.SetKeySchema(f3)
	}
	if r.ko.Spec.LocalSecondaryIndexes != nil {
		f4 := []*svcsdk.LocalSecondaryIndex{}
		for _, f4iter := range r.ko.Spec.LocalSecondaryIndexes {
			f4elem := &svcsdk.LocalSecondaryIndex{}
			if f4iter.IndexName != nil {
				f4elem.SetIndexName(*f4iter.IndexName)
			}
			if f4iter.KeySchema != nil {
				f4elemf1 := []*svcsdk.KeySchemaElement{}
				for _, f4elemf1iter := range f4iter.KeySchema {
					f4elemf1elem := &svcsdk.KeySchemaElement{}
					if f4elemf1iter.AttributeName != nil {
						f4elemf1elem.SetAttributeName(*f4elemf1iter.AttributeName)
					}
					if f4elemf1iter.KeyType != nil {
						f4elemf1elem.SetKeyType(*f4elemf1iter.KeyType)
					}
					f4elemf1 = append(f4elemf1, f4elemf1elem)
				}
				f4elem.SetKeySchema(f4elemf1)
			}
			if f4iter.Projection != nil {
				f4elemf2 := &svcsdk.Projection{}
				if f4iter.Projection.NonKeyAttributes != nil {
					f4elemf2f0 := []*string{}
					for _, f4elemf2f0iter := range f4iter.Projection.NonKeyAttributes {
						var f4elemf2f0elem string
						f4elemf2f0elem = *f4elemf2f0iter
						f4elemf2f0 = append(f4elemf2f0, &f4elemf2f0elem)
					}
					f4elemf2.SetNonKeyAttributes(f4elemf2f0)
				}
				if f4iter.Projection.ProjectionType != nil {
					f4elemf2.SetProjectionType(*f4iter.Projection.ProjectionType)
				}
				f4elem.SetProjection(f4elemf2)
			}
			f4 = append(f4, f4elem)
		}
		res.SetLocalSecondaryIndexes(f4)
	}
	if r.ko.Spec.ProvisionedThroughput != nil {
		f5 := &svcsdk.ProvisionedThroughput{}
		if r.ko.Spec.ProvisionedThroughput.ReadCapacityUnits != nil {
			f5.SetReadCapacityUnits(*r.ko.Spec.ProvisionedThroughput.ReadCapacityUnits)
		}
		if r.ko.Spec.ProvisionedThroughput.WriteCapacityUnits != nil {
			f5.SetWriteCapacityUnits(*r.ko.Spec.ProvisionedThroughput.WriteCapacityUnits)
		}
		res.SetProvisionedThroughput(f5)
	}
	if r.ko.Spec.SSESpecification != nil {
		f6 := &svcsdk.SSESpecification{}
		if r.ko.Spec.SSESpecification.Enabled != nil {
			f6.SetEnabled(*r.ko.Spec.SSESpecification.Enabled)
		}
		if r.ko.Spec.SSESpecification.KMSMasterKeyID != nil {
			f6.SetKMSMasterKeyId(*r.ko.Spec.SSESpecification.KMSMasterKeyID)
		}
		if r.ko.Spec.SSESpecification.SSEType != nil {
			f6.SetSSEType(*r.ko.Spec.SSESpecification.SSEType)
		}
		res.SetSSESpecification(f6)
	}
	if r.ko.Spec.StreamSpecification != nil {
		f7 := &svcsdk.StreamSpecification{}
		if r.ko.Spec.StreamSpecification.StreamEnabled != nil {
			f7.SetStreamEnabled(*r.ko.Spec.StreamSpecification.StreamEnabled)
		}
		if r.ko.Spec.StreamSpecification.StreamViewType != nil {
			f7.SetStreamViewType(*r.ko.Spec.StreamSpecification.StreamViewType)
		}
		res.SetStreamSpecification(f7)
	}
	if r.ko.Spec.TableName != nil {
		res.SetTableName(*r.ko.Spec.TableName)
	}
	if r.ko.Spec.Tags != nil {
		f9 := []*svcsdk.Tag{}
		for _, f9iter := range r.ko.Spec.Tags {
			f9elem := &svcsdk.Tag{}
			if f9iter.Key != nil {
				f9elem.SetKey(*f9iter.Key)
			}
			if f9iter.Value != nil {
				f9elem.SetValue(*f9iter.Value)
			}
			f9 = append(f9, f9elem)
		}
		res.SetTags(f9)
	}
`
	assert.Equal(expCreateInput, crd.GoCodeSetInput(model.OpTypeCreate, "r.ko", "res", 1))

	expStatusFieldCamel := []string{
		"ArchivalSummary",
		"BillingModeSummary",
		"CreationDateTime",
		"GlobalTableVersion",
		"ItemCount",
		"LatestStreamARN",
		"LatestStreamLabel",
		"Replicas",
		"RestoreSummary",
		"SSEDescription",
		"TableID",
		"TableSizeBytes",
		"TableStatus",
	}
	assert.Equal(expStatusFieldCamel, attrCamelNames(statusFields))

	// The DynamoDB API uses an API that uses "wrapper" single-member objects
	// in the JSON response for the create/describe calls. In other words, the
	// returned result from the DescribeTable API looks like this:
	//
	// {
	//   "table": {
	//	 .. bunch of fields for the table ..
	//   }
	// }
	//
	// However, the *ShapeName* of the "table" field is actually
	// TableDescription. This tests that we're properly outputting the
	// memberName (which is "Table" and not "TableDescription") when we build
	// the Table CRD's Status field from the DescribeTableOutput shape.
	expReadOneOutput := `
	if resp.Table.ArchivalSummary != nil {
		f0 := &svcapitypes.ArchivalSummary{}
		if resp.Table.ArchivalSummary.ArchivalBackupArn != nil {
			f0.ArchivalBackupARN = resp.Table.ArchivalSummary.ArchivalBackupArn
		}
		if resp.Table.ArchivalSummary.ArchivalDateTime != nil {
			f0.ArchivalDateTime = &metav1.Time{*resp.Table.ArchivalSummary.ArchivalDateTime}
		}
		if resp.Table.ArchivalSummary.ArchivalReason != nil {
			f0.ArchivalReason = resp.Table.ArchivalSummary.ArchivalReason
		}
		ko.Status.ArchivalSummary = f0
	}
	if resp.Table.BillingModeSummary != nil {
		f2 := &svcapitypes.BillingModeSummary{}
		if resp.Table.BillingModeSummary.BillingMode != nil {
			f2.BillingMode = resp.Table.BillingModeSummary.BillingMode
		}
		if resp.Table.BillingModeSummary.LastUpdateToPayPerRequestDateTime != nil {
			f2.LastUpdateToPayPerRequestDateTime = &metav1.Time{*resp.Table.BillingModeSummary.LastUpdateToPayPerRequestDateTime}
		}
		ko.Status.BillingModeSummary = f2
	}
	if resp.Table.CreationDateTime != nil {
		ko.Status.CreationDateTime = &metav1.Time{*resp.Table.CreationDateTime}
	}
	if resp.Table.GlobalTableVersion != nil {
		ko.Status.GlobalTableVersion = resp.Table.GlobalTableVersion
	}
	if resp.Table.ItemCount != nil {
		ko.Status.ItemCount = resp.Table.ItemCount
	}
	if resp.Table.LatestStreamArn != nil {
		ko.Status.LatestStreamARN = resp.Table.LatestStreamArn
	}
	if resp.Table.LatestStreamLabel != nil {
		ko.Status.LatestStreamLabel = resp.Table.LatestStreamLabel
	}
	if resp.Table.Replicas != nil {
		f12 := []*svcapitypes.ReplicaDescription{}
		for _, f12iter := range resp.Table.Replicas {
			f12elem := &svcapitypes.ReplicaDescription{}
			if f12iter.GlobalSecondaryIndexes != nil {
				f12elemf0 := []*svcapitypes.ReplicaGlobalSecondaryIndexDescription{}
				for _, f12elemf0iter := range f12iter.GlobalSecondaryIndexes {
					f12elemf0elem := &svcapitypes.ReplicaGlobalSecondaryIndexDescription{}
					if f12elemf0iter.IndexName != nil {
						f12elemf0elem.IndexName = f12elemf0iter.IndexName
					}
					if f12elemf0iter.ProvisionedThroughputOverride != nil {
						f12elemf0elemf1 := &svcapitypes.ProvisionedThroughputOverride{}
						if f12elemf0iter.ProvisionedThroughputOverride.ReadCapacityUnits != nil {
							f12elemf0elemf1.ReadCapacityUnits = f12elemf0iter.ProvisionedThroughputOverride.ReadCapacityUnits
						}
						f12elemf0elem.ProvisionedThroughputOverride = f12elemf0elemf1
					}
					f12elemf0 = append(f12elemf0, f12elemf0elem)
				}
				f12elem.GlobalSecondaryIndexes = f12elemf0
			}
			if f12iter.KMSMasterKeyId != nil {
				f12elem.KMSMasterKeyID = f12iter.KMSMasterKeyId
			}
			if f12iter.ProvisionedThroughputOverride != nil {
				f12elemf2 := &svcapitypes.ProvisionedThroughputOverride{}
				if f12iter.ProvisionedThroughputOverride.ReadCapacityUnits != nil {
					f12elemf2.ReadCapacityUnits = f12iter.ProvisionedThroughputOverride.ReadCapacityUnits
				}
				f12elem.ProvisionedThroughputOverride = f12elemf2
			}
			if f12iter.RegionName != nil {
				f12elem.RegionName = f12iter.RegionName
			}
			if f12iter.ReplicaStatus != nil {
				f12elem.ReplicaStatus = f12iter.ReplicaStatus
			}
			if f12iter.ReplicaStatusDescription != nil {
				f12elem.ReplicaStatusDescription = f12iter.ReplicaStatusDescription
			}
			if f12iter.ReplicaStatusPercentProgress != nil {
				f12elem.ReplicaStatusPercentProgress = f12iter.ReplicaStatusPercentProgress
			}
			f12 = append(f12, f12elem)
		}
		ko.Status.Replicas = f12
	}
	if resp.Table.RestoreSummary != nil {
		f13 := &svcapitypes.RestoreSummary{}
		if resp.Table.RestoreSummary.RestoreDateTime != nil {
			f13.RestoreDateTime = &metav1.Time{*resp.Table.RestoreSummary.RestoreDateTime}
		}
		if resp.Table.RestoreSummary.RestoreInProgress != nil {
			f13.RestoreInProgress = resp.Table.RestoreSummary.RestoreInProgress
		}
		if resp.Table.RestoreSummary.SourceBackupArn != nil {
			f13.SourceBackupARN = resp.Table.RestoreSummary.SourceBackupArn
		}
		if resp.Table.RestoreSummary.SourceTableArn != nil {
			f13.SourceTableARN = resp.Table.RestoreSummary.SourceTableArn
		}
		ko.Status.RestoreSummary = f13
	}
	if resp.Table.SSEDescription != nil {
		f14 := &svcapitypes.SSEDescription{}
		if resp.Table.SSEDescription.InaccessibleEncryptionDateTime != nil {
			f14.InaccessibleEncryptionDateTime = &metav1.Time{*resp.Table.SSEDescription.InaccessibleEncryptionDateTime}
		}
		if resp.Table.SSEDescription.KMSMasterKeyArn != nil {
			f14.KMSMasterKeyARN = resp.Table.SSEDescription.KMSMasterKeyArn
		}
		if resp.Table.SSEDescription.SSEType != nil {
			f14.SSEType = resp.Table.SSEDescription.SSEType
		}
		if resp.Table.SSEDescription.Status != nil {
			f14.Status = resp.Table.SSEDescription.Status
		}
		ko.Status.SSEDescription = f14
	}
	if resp.Table.TableId != nil {
		ko.Status.TableID = resp.Table.TableId
	}
	if resp.Table.TableSizeBytes != nil {
		ko.Status.TableSizeBytes = resp.Table.TableSizeBytes
	}
	if resp.Table.TableStatus != nil {
		ko.Status.TableStatus = resp.Table.TableStatus
	}
`
	assert.Equal(expReadOneOutput, crd.GoCodeSetOutput(model.OpTypeGet, "resp", "ko.Status", 1))
}
