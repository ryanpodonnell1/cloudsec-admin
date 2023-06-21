package main

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/guardduty"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/gin-gonic/gin"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	r := gin.Default()
	defer r.Run("localhost:8080")
	g := r.Group("/api/v1/")

	aws := g.Group("/aws")
	gdRoute := aws.Group("/guardduty/")

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return err
	}

	gdRoute.GET("/status", func(c *gin.Context) {
		c.DefaultQuery("region", "us-west-2")
		s := GetGuardDutyStatus(c, context.Background(), cfg)
		c.JSON(200, s)
	})

	return nil
}

type guardDutyAccountStatus map[string]guardDutyDetectorStatus

type guardDutyDetectorStatus struct {
	Detector string
	Status   string
	Region   string
	Err      error `json:",omitempty"`
}

func GetGuardDutyStatus(c *gin.Context, ctx context.Context, cfg aws.Config) guardDutyAccountStatus {
	m := guardDutyAccountStatus{}

	st := sts.NewFromConfig(cfg)
	region := c.Query("region")
	s := getGuardDutyStatus(context.TODO(), guardduty.NewFromConfig(cfg, func(o *guardduty.Options) {
		o.Region = region
	}))

	s.Region = region
	id, err := st.GetCallerIdentity(ctx, nil)
	if err != nil {
		s.Err = err
		m["UNKNOWN"] = s
	} else {
		m[*id.Account] = s
	}
	return m
}

func getGuardDutyStatus(ctx context.Context, c *guardduty.Client) guardDutyDetectorStatus {
	m := guardDutyDetectorStatus{}
	o, err := c.ListDetectors(ctx, &guardduty.ListDetectorsInput{})
	if err != nil {
		m.Err = err
		return m
	}

	if len(o.DetectorIds) == 0 {
		m.Detector = "NONE_CONFIGURED"
		m.Status = "N/A"
	}

	for _, v := range o.DetectorIds {
		status, err := c.GetDetector(ctx, &guardduty.GetDetectorInput{
			DetectorId: &v,
		})
		if err != nil {
			m.Err = err
			continue
		}
		m.Detector = v
		m.Status = string(status.Status)

	}
	return m
}
