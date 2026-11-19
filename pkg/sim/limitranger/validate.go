package limitranger

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

type observed struct {
	request, limit, enforced int64
}

func observe(request, limit, enforced resource.Quantity) observed {
	o := observed{request: request.Value(), limit: limit.Value(), enforced: enforced.Value()}
	if o.request <= resource.MaxMilliValue && o.limit <= resource.MaxMilliValue && o.enforced <= resource.MaxMilliValue {
		o = observed{request: request.MilliValue(), limit: limit.MilliValue(), enforced: enforced.MilliValue()}
	}
	return o
}

func minConstraint(limitType, name string, enforced resource.Quantity, requests, limits corev1.ResourceList) error {
	req, reqExists := quantityOf(requests, corev1.ResourceName(name))
	lim, limExists := quantityOf(limits, corev1.ResourceName(name))
	o := observe(req, lim, enforced)
	switch {
	case !reqExists:
		return fmt.Errorf("minimum %s usage per %s is %s.  No request is specified", name, limitType, enforced.String())
	case o.request < o.enforced:
		return fmt.Errorf("minimum %s usage per %s is %s, but request is %s", name, limitType, enforced.String(), req.String())
	case limExists && o.limit < o.enforced:
		return fmt.Errorf("minimum %s usage per %s is %s, but limit is %s", name, limitType, enforced.String(), lim.String())
	}
	return nil
}

func maxConstraint(limitType, name string, enforced resource.Quantity, requests, limits corev1.ResourceList) error {
	req, reqExists := quantityOf(requests, corev1.ResourceName(name))
	lim, limExists := quantityOf(limits, corev1.ResourceName(name))
	o := observe(req, lim, enforced)
	switch {
	case !limExists:
		return fmt.Errorf("maximum %s usage per %s is %s.  No limit is specified", name, limitType, enforced.String())
	case o.limit > o.enforced:
		return fmt.Errorf("maximum %s usage per %s is %s, but limit is %s", name, limitType, enforced.String(), lim.String())
	case reqExists && o.request > o.enforced:
		return fmt.Errorf("maximum %s usage per %s is %s, but request is %s", name, limitType, enforced.String(), req.String())
	}
	return nil
}

func ratioConstraint(limitType, name string, enforced resource.Quantity, requests, limits corev1.ResourceList) error {
	req, reqExists := quantityOf(requests, corev1.ResourceName(name))
	lim, limExists := quantityOf(limits, corev1.ResourceName(name))
	o := observe(req, lim, enforced)
	if !reqExists || o.request == 0 {
		return fmt.Errorf("%s max limit to request ratio per %s is %s, but no request is specified or request is 0", name, limitType, enforced.String())
	}
	if !limExists || o.limit == 0 {
		return fmt.Errorf("%s max limit to request ratio per %s is %s, but no limit is specified or limit is 0", name, limitType, enforced.String())
	}
	ratio := float64(o.limit) / float64(o.request)
	shown, allowed := ratio, float64(enforced.Value())
	if enforced.Value() <= resource.MaxMilliValue {
		ratio, allowed = ratio*1000, float64(enforced.MilliValue())
	}
	if ratio > allowed {
		return fmt.Errorf("%s max limit to request ratio per %s is %s, but provided ratio is %f", name, limitType, enforced.String(), shown)
	}
	return nil
}

func sortedNames(list corev1.ResourceList) []string {
	out := make([]string, 0, len(list))
	for name := range list {
		out = append(out, string(name))
	}
	sort.Strings(out)
	return out
}

func checkItem(item corev1.LimitRangeItem, requests, limits corev1.ResourceList) []error {
	limitType := string(item.Type)
	var errs []error
	for _, name := range sortedNames(item.Min) {
		errs = appendErr(errs, minConstraint(limitType, name, item.Min[corev1.ResourceName(name)], requests, limits))
	}
	for _, name := range sortedNames(item.Max) {
		errs = appendErr(errs, maxConstraint(limitType, name, item.Max[corev1.ResourceName(name)], requests, limits))
	}
	for _, name := range sortedNames(item.MaxLimitRequestRatio) {
		errs = appendErr(errs, ratioConstraint(limitType, name, item.MaxLimitRequestRatio[corev1.ResourceName(name)], requests, limits))
	}
	return errs
}

func appendErr(errs []error, err error) []error {
	if err == nil {
		return errs
	}
	return append(errs, err)
}

func validateOne(pod *corev1.Pod, lr *corev1.LimitRange) error {
	var errs []error
	for _, item := range lr.Spec.Limits {
		switch item.Type {
		case corev1.LimitTypeContainer:
			for _, c := range append(append([]corev1.Container(nil), pod.Spec.Containers...), pod.Spec.InitContainers...) {
				errs = append(errs, checkItem(item, c.Resources.Requests, c.Resources.Limits)...)
			}
		case corev1.LimitTypePod:
			requests, limits := podTotals(pod)
			errs = append(errs, checkItem(item, requests, limits)...)
		}
	}
	return aggregate(errs)
}

func aggregate(errs []error) error {
	switch len(errs) {
	case 0:
		return nil
	case 1:
		return errs[0]
	}
	msgs := make([]string, 0, len(errs))
	for _, err := range errs {
		msgs = append(msgs, err.Error())
	}
	return errors.New("[" + strings.Join(msgs, ", ") + "]")
}

func Validate(pod *corev1.Pod, ranges []*corev1.LimitRange) error {
	for _, lr := range ranges {
		if err := validateOne(pod, lr); err != nil {
			return err
		}
	}
	return nil
}
