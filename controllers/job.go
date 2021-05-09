package controllers

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-logr/logr"

	traitv1 "github.com/ghostbaby/cfs-trait/api/v1"

	"github.com/ghostbaby/cfs-trait/controllers/base"
)

// jobs
type Jobs struct {
	Client     base.BaseClient
	Node       string
	RetryTimes int
}

type Results struct {
	Resp error
	Node string
}

const (
	DefaultThreadPoolSize = 30
	DefaultRetryTimes     = 3
)

func ExecWork(cfs *traitv1.CfsTrait, jobs <-chan *Jobs, results chan<- *Results, log logr.Logger, force bool) {

	for j := range jobs {
		var res Results
		rt := j.RetryTimes
		res.Node = j.Node
		for {
			var query base.QueryResult

			res.Resp = j.Client.Post(context.TODO(), "/api/v1/cfs/config", cfs, &query)
			if res.Resp != nil {
				rt--
				if rt == 0 {
					results <- &res
					break
				}
				continue
			}

			if len(query.Result) == 0 {
				rt--
				if rt == 0 {
					results <- &res
					break
				}
				continue
			}

			if cfs.Spec.Period == query.Result[0].CfsPeriodUS && cfs.Spec.Quota == query.Result[0].CfsQuotaUS {
				log.Info("cfs config not changed, skip to exec",
					"node", j.Node, "namespace", cfs.GetNamespace(), "name", cfs.GetName())
				results <- &res
				break
			}

			if !force {

				expect := float64(cfs.Spec.Quota) / float64(cfs.Spec.Period)
				current := float64(query.Result[0].CfsQuotaUS) / float64(query.Result[0].CfsPeriodUS)

				if expect < current {
					log.Info("fail to check expect cfs config, skip to exec",
						"expect", expect, "current", current)
					res.Resp = errors.New(
						fmt.Sprintf("fail to check expect cfs config, expect %f,current %f, skip to exec",
							expect, current),
					)

					rt--
					if rt == 0 {
						results <- &res
						break
					}
					continue
				}
			}

			res.Resp = j.Client.Post(context.TODO(), "/api/v1/cfs/broker", cfs, nil)
			if res.Resp != nil {
				rt--
				if rt == 0 {
					results <- &res
					break
				}
				continue
			}
			results <- &res
			break
		}
	}
}
