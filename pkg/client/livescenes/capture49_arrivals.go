package livescenes

import (
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	app "github.com/waxb/kubernites/pkg/client/app"
	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/world"
	"github.com/waxb/kubernites/pkg/live"
)

const (
	arrivalNS    = "shop"
	arrivalClass = "nginx"
)

func init() {
	app.CaptureScenes["live-listed"] = listedScene(KindSnapshot, func(live.Snapshot) []live.Listed { return []live.Listed{{List: "pods"}} })
	app.CaptureScenes["live-listed-cap"] = listedScene(ManagedSnapshot, func(s live.Snapshot) []live.Listed {
		out := []live.Listed{{List: "pods"}}
		for _, c := range s.CRDs[len(s.CRDs)-3:] {
			out = append(out, live.Listed{List: c.List(), PastCap: true})
		}
		return out
	})
	app.CaptureScenes["live-new-service"] = arrivalScene("Service", "basket", ArrivingService())
	app.CaptureScenes["live-new-ingress"] = arrivalScene("Ingress", "basket", ArrivingService(), ArrivingIngress())
	app.CaptureScenes["live-new-gateway"] = arrivalScene("Gateway", "front", ArrivingGateway())
	for _, name := range []string{"live-listed", "live-listed-cap", "live-new-service", "live-new-ingress", "live-new-gateway"} {
		app.LiveScenes[name] = true
	}
}

func listedScene(snapshot func() live.Snapshot, listed func(live.Snapshot) []live.Listed) func(*app.Game) hud.Input {
	return func(g *app.Game) hud.Input {
		snap := snapshot()
		r := openLive(g, snap)
		if err := r.ReplayListed(listed(snap)); err != nil {
			panic(err)
		}
		s := g.Sess
		Settle(s, 1)
		s.CenterOn(float64(s.Site.Gate.X), float64(s.Site.Gate.Y))
		return hoverButton(s, "info:live:listed")
	}
}

func ArrivingService() Replayed {
	return Replayed{List: "services", Type: "ADDED", Object: corev1.Service{
		TypeMeta:   metav1.TypeMeta{Kind: "Service", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{Name: "basket", Namespace: arrivalNS, UID: "uid-arriving-basket", ResourceVersion: "5001"},
		Spec:       corev1.ServiceSpec{Type: corev1.ServiceTypeClusterIP, ClusterIP: "10.96.200.17", Ports: []corev1.ServicePort{{Port: 80}}}}}
}

func ArrivingIngress() Replayed {
	class := arrivalClass
	return Replayed{List: "ingresses", Type: "ADDED", Object: networkingv1.Ingress{
		TypeMeta:   metav1.TypeMeta{Kind: "Ingress", APIVersion: "networking.k8s.io/v1"},
		ObjectMeta: metav1.ObjectMeta{Name: "basket", Namespace: arrivalNS, UID: "uid-arriving-basket-ing", ResourceVersion: "5002"},
		Spec: networkingv1.IngressSpec{IngressClassName: &class, Rules: []networkingv1.IngressRule{{Host: "basket.shop.example", IngressRuleValue: networkingv1.IngressRuleValue{
			HTTP: &networkingv1.HTTPIngressRuleValue{Paths: []networkingv1.HTTPIngressPath{{Path: "/", Backend: networkingv1.IngressBackend{Service: &networkingv1.IngressServiceBackend{Name: "basket"}}}}}}}}}}}
}

func ArrivingGateway() Replayed {
	return Replayed{List: "gateways", Type: "ADDED", Object: k8sGateway()}
}

func k8sGateway() live.Gateway {
	return live.Gateway{
		ObjectMeta: metav1.ObjectMeta{Name: "front", Namespace: arrivalNS, UID: "uid-arriving-front", ResourceVersion: "5003"},
		Spec:       live.GatewaySpec{GatewayClassName: "example", Listeners: []live.GatewayListener{{Name: "http", Port: 80, Protocol: "HTTP"}}}}
}

func arrivalScene(kind, name string, events ...Replayed) func(*app.Game) hud.Input {
	return func(g *app.Game) hud.Input {
		r := openLive(g, KindSnapshot())
		s := g.Sess
		Settle(s, 1)
		if err := Play(r, events); err != nil {
			panic(err)
		}
		Settle(s, 1)
		for _, st := range s.Site.Structures {
			if st.Object.Kind != kind || st.Object.Namespace != arrivalNS || st.Object.Name != name {
				continue
			}
			c := world.FootprintCentre(st)
			s.CenterOn(c.X, c.Y)
			x, y := world.ScreenOf(s.Cam, s.Site, c)
			return hud.Input{MX: int(x), MY: int(y) - hoverLift*int(s.Cam.Scale())}
		}
		panic("the arriving " + kind + " was not drawn")
	}
}
