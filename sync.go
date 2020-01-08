package main

import (
	"log"
	"sort"
	"strings"
	"strconv"

	cloudflare "github.com/cloudflare/cloudflare-go"
	"github.com/pkg/errors"
)

func sync(ips []string, dnsNames []string, cloudflareTTL int, cloudflareProxy bool) error {
	//Should make lbAddressLimit a Configuration property in the secret
	//Fact is I didn't want to pay extra for 3 addresses on cloudflare when only 2 are included.
	lbAddressLimit := 2
	sort.Strings(ips)
	nodeIPs := append(ips[0:lbAddressLimit])

	api, err := cloudflare.New(options.CloudflareAPIKey, options.CloudflareAPIEmail)
	if err != nil {
		return errors.Wrap(err, "failed to access cloudflare api")
	}


	pools, err := api.ListLoadBalancerPools()
	if err != nil {
		return errors.Wrap(err, "failed to list the load balancer pools")
	}
	//Should probably add a conditional on the name of the pool so 
	//we're not updating every single pool to our kubernetes node ips 😬
	//I currently only have 1 so this is fine.
	for _, pool := range pools {
		lbIps := []string{}
		for _, origin := range pool.Origins{
			lbIps = append(lbIps, origin.Address)
		}
		sort.Strings(lbIps)

		nodeIPsJoin := strings.Join(nodeIPs, ",")
		lbIPsJoin := strings.Join(lbIps, ",")
		
		if nodeIPsJoin == lbIPsJoin {
			log.Printf("no change detected")
			return nil
		}

		log.Printf("node-ips: %s", nodeIPsJoin)
		log.Printf("loadbalancer-ips: %s", lbIPsJoin)

		newOrigins := []cloudflare.LoadBalancerOrigin{}
		for index, ip := range nodeIPs {
			origin := cloudflare.LoadBalancerOrigin{
				Name: "kube-"+strconv.Itoa(index),
				Address: string(ip),
				Enabled: true,
				Weight: 1,
			}
			newOrigins = append(newOrigins, origin)
		}

		log.Printf("Updating Loadbalancer pool %s",pool.Name)
		log.Println(newOrigins)
		pool.Origins = newOrigins
		_, err := api.ModifyLoadBalancerPool(pool)
		if err != nil {
			errors.Wrapf(err, "Error updating the Load Balaner Pool %s", pool.Name)
		}
	}

	return nil
}
