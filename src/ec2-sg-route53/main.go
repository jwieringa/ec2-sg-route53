package main

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/route53"
	"os"
)

type Config struct {
	root_zone   string
	fqdn        string
	sg_name     string
	weight      int64
	priority    int64
	record_ttl  int64
	record_type string
	port        int64
}

var conf Config

func check(err error) {
	if err != nil {
		log.WithFields(log.Fields{"msg": err}).Fatal()
	}
	return
}

func fetchHostedZoneId(root_zone string) *string {
	sess, err := session.NewSession()
	check(err)
	svc := route53.New(sess)
	params := &route53.ListHostedZonesByNameInput{
		DNSName: aws.String(root_zone),
	}
	resp, err := svc.ListHostedZonesByName(params)
	check(err)

	if resp.HostedZones[0] == nil {
		log.WithFields(log.Fields{
			"msg": fmt.Sprintf("No zone id was found for %v", root_zone),
		}).Fatal()
	}
	log.WithFields(log.Fields{
		"msg": fmt.Sprintf("Fectched Zone ID %v for DNS name %s", *resp.HostedZones[0].Id, *resp.HostedZones[0].Name),
	}).Info()
	return resp.HostedZones[0].Id
}

func ec2PrivateIps(sg string) (ips []*string) {
	sess, err := session.NewSession()
	svc := ec2.New(sess)
	params := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("instance.group-name"),
				Values: []*string{
					aws.String(sg),
				},
			},
			{
				Name: aws.String("instance-state-name"),
				Values: []*string{
					aws.String("running"),
					aws.String("pending"),
				},
			},
		},
	}
	log.WithFields(log.Fields{
		"msg": fmt.Sprintf("Fetching list of instances in security group %v", sg)},
	).Info()
	resp, err := svc.DescribeInstances(params)
	check(err)

	if len(resp.Reservations) == 0 {
		log.WithFields(log.Fields{"msg": "There are no isntances in the security group"}).Fatal()
	}

	for _, reservation := range resp.Reservations {
		for _, instance := range reservation.Instances {
			log.WithFields(log.Fields{
				"msg": fmt.Sprintf("Adding private IP %v", *instance.PrivateIpAddress)},
			).Info()
			ips = append(ips, instance.PrivateIpAddress)
		}
	}
	return ips
}

func formatSrvRecords(ips []*string) []*string {
	for i, ip := range ips {
		srv := fmt.Sprintf("%d %d %d %s", conf.priority, conf.weight, conf.port, *ip)
		ips[i] = &srv
	}
	return ips
}

func createResourceRecords(records []*string) (resourceRecords []*route53.ResourceRecord) {
	for _, rec := range records {
		resourceRecords = append(resourceRecords, &route53.ResourceRecord{
			Value: aws.String(*rec),
		})
	}
	return resourceRecords
}

func updateRoute53(resourceRecords []*route53.ResourceRecord, zone_id *string) {
	sess, err := session.NewSession()
	check(err)
	svc := route53.New(sess)
	params := &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(*zone_id),
		ChangeBatch: &route53.ChangeBatch{
			Changes: []*route53.Change{
				{
					Action: aws.String("UPSERT"),
					ResourceRecordSet: &route53.ResourceRecordSet{
						Name:            aws.String(conf.fqdn),
						Type:            aws.String(conf.record_type),
						TTL:             aws.Int64(conf.record_ttl),
						ResourceRecords: resourceRecords,
					},
				},
			},
		},
	}
	resp, err := svc.ChangeResourceRecordSets(params)
	check(err)
	log.WithFields(log.Fields{
		"msg": fmt.Sprintf("Updated %v and status is %s", conf.fqdn, *resp.ChangeInfo.Status),
	}).Info()
}

func main() {
	conf.root_zone = os.Getenv("ROOT_ZONE")
	conf.fqdn = os.Getenv("FQDN")
	conf.sg_name = os.Getenv("SECURITY_GROUP_NAME")
	conf.weight = 0
	conf.priority = 0
	conf.record_ttl = 180
	conf.record_type = "SRV"
	conf.port = 2380

	private_ips := ec2PrivateIps(conf.sg_name)
	records := formatSrvRecords(private_ips)
	updateRoute53(createResourceRecords(records), fetchHostedZoneId(conf.root_zone))
}
