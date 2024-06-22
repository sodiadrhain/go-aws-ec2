package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func main() {
	var (
		insatnceId string
		err        error
	)

	region := "eu-west-2"

	insatnceId, err = createEc2(context.Background(), region)
	if err != nil {
		log.Fatalf("Error in %s\n", err)
	}

	fmt.Printf("insatnceId: %s\n", insatnceId)
}

func createEc2(ctx context.Context, region string) (string, error) {
	// Load AWS Configuration from (~/.aws/config)
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion((region)))
	if err != nil {
		return "", fmt.Errorf("LoadDefaultConfig: %s", err)
	}

	// Create an Amazon EC2 service client
	ec2Client := ec2.NewFromConfig(cfg)
	keyPairName := aws.String("go-aws-ec2")

	ec2Client, err = createKeyPair(keyPairName, ctx, ec2Client)
	if err != nil {
		return "", err
	}

	describeImages, err := ec2Client.DescribeImages(ctx, &ec2.DescribeImagesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("name"),
				Values: []string{"ubuntu/images/hvm-ssd-gp3/ubuntu-noble-24.04-amd64-server-20240423"},
			},
			{
				Name:   aws.String("virtualization-type"),
				Values: []string{"hvm"},
			},
		},
		Owners: []string{"099720109477"},
	})
	if err != nil {
		return "", fmt.Errorf("DescribeImages: %s", err)
	}

	if len(describeImages.Images) == 0 {
		return "", fmt.Errorf("DescribeImages has empty length, %d", len(describeImages.Images))
	}

	instance, err := ec2Client.RunInstances(ctx, &ec2.RunInstancesInput{
		MaxCount:     aws.Int32(1),
		MinCount:     aws.Int32(1),
		ImageId:      describeImages.Images[0].ImageId,
		KeyName:      keyPairName,
		InstanceType: types.InstanceTypeT2Micro,
	})
	if err != nil {
		return "", fmt.Errorf("RunInstances: %s", err)
	}

	if len(instance.Instances) == 0 {
		return "", fmt.Errorf("instances has empty length, %d", len(instance.Instances))
	}

	return *instance.Instances[0].InstanceId, nil
}

func createKeyPair(keyName *string, ctx context.Context, client *ec2.Client) (*ec2.Client, error) {

	// Check if key pair exists
	keyPairs, err := client.DescribeKeyPairs(ctx, &ec2.DescribeKeyPairsInput{
		KeyNames: []string{*keyName},
	})
	if err != nil && !strings.Contains(err.Error(), "InvalidKeyPair.NotFound:") {
		return nil, fmt.Errorf("DescribeKeyPairs: %s", err)
	}

	if keyPairs == nil || len(keyPairs.KeyPairs) == 0 {
		// Create Key pair
		keyPair, err := client.CreateKeyPair(ctx, &ec2.CreateKeyPairInput{
			KeyName: keyName,
		})
		if err != nil {
			return nil, fmt.Errorf("CreateKeyPair: %s", err)
		}

		// Save key pair
		err = os.WriteFile("go-aws-ec2.pem", []byte(*keyPair.KeyMaterial), 0600)
		if err != nil {
			return nil, fmt.Errorf("WriteFile (save key pair): %s", err)
		}

		fmt.Printf("Key pair saved, go-aws-ec2.pem \n")
	}

	return client, nil
}
