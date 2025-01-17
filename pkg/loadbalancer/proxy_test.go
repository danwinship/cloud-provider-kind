package loadbalancer

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func makeNode(name string, ip string) *v1.Node {
	return &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1.NodeSpec{},
		Status: v1.NodeStatus{
			Addresses: []v1.NodeAddress{
				{Type: v1.NodeInternalIP, Address: ip},
			},
		},
	}
}

func makeService(name string) *v1.Service {
	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1.ServiceSpec{
			Type: v1.ServiceTypeClusterIP,
			Ports: []v1.ServicePort{
				{Port: 80},
			},
		},
	}
}

func Test_generateConfig(t *testing.T) {
	tests := []struct {
		name    string
		service *v1.Service
		nodes   []*v1.Node
		want    *proxyConfigData
	}{
		{
			name: "empty",
		},
		{
			name: "simple service",
			service: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: v1.ServiceSpec{
					Type:                  v1.ServiceTypeLoadBalancer,
					ExternalTrafficPolicy: v1.ServiceExternalTrafficPolicyLocal,
					IPFamilies:            []v1.IPFamily{v1.IPv4Protocol},
					Ports: []v1.ServicePort{
						{
							Port:       80,
							TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: 8080},
							NodePort:   30000,
							Protocol:   v1.ProtocolTCP,
						},
					},
					HealthCheckNodePort: 32000,
				},
			},
			nodes: []*v1.Node{
				makeNode("a", "10.0.0.1"),
				makeNode("b", "10.0.0.2"),
			},
			want: &proxyConfigData{
				HealthCheckPort: 32000,
				ServicePorts: map[string]servicePort{
					"IPv4_80_TCP": servicePort{
						Listener: endpoint{Address: "0.0.0.0", Port: 80, Protocol: string(v1.ProtocolTCP)},
						Cluster:  []endpoint{{"10.0.0.1", 30000, string(v1.ProtocolTCP)}, {"10.0.0.2", 30000, string(v1.ProtocolTCP)}},
					},
				},
			},
		},
		{
			name: "multiport service",
			service: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: v1.ServiceSpec{
					Type:                  v1.ServiceTypeLoadBalancer,
					ExternalTrafficPolicy: v1.ServiceExternalTrafficPolicyLocal,
					IPFamilies:            []v1.IPFamily{v1.IPv4Protocol},
					Ports: []v1.ServicePort{
						{
							Port:       80,
							TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: 8080},
							NodePort:   30000,
							Protocol:   v1.ProtocolTCP,
						},
						{
							Port:       443,
							TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: 8080},
							NodePort:   31000,
							Protocol:   v1.ProtocolTCP,
						},
					},
					HealthCheckNodePort: 32000,
				},
			},
			nodes: []*v1.Node{
				makeNode("a", "10.0.0.1"),
				makeNode("b", "10.0.0.2"),
			},
			want: &proxyConfigData{
				HealthCheckPort: 32000,
				ServicePorts: map[string]servicePort{
					"IPv4_80_TCP": servicePort{
						Listener: endpoint{Address: "0.0.0.0", Port: 80, Protocol: string(v1.ProtocolTCP)},
						Cluster:  []endpoint{{"10.0.0.1", 30000, string(v1.ProtocolTCP)}, {"10.0.0.2", 30000, string(v1.ProtocolTCP)}},
					},
					"IPv4_443_TCP": servicePort{
						Listener: endpoint{Address: "0.0.0.0", Port: 443, Protocol: string(v1.ProtocolTCP)},
						Cluster:  []endpoint{{"10.0.0.1", 31000, string(v1.ProtocolTCP)}, {"10.0.0.2", 31000, string(v1.ProtocolTCP)}},
					},
				},
			},
		},
		{
			name: "multiport different protocol service",
			service: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: v1.ServiceSpec{
					Type:                  v1.ServiceTypeLoadBalancer,
					ExternalTrafficPolicy: v1.ServiceExternalTrafficPolicyLocal,
					IPFamilies:            []v1.IPFamily{v1.IPv4Protocol},
					Ports: []v1.ServicePort{
						{
							Port:       80,
							TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: 8080},
							NodePort:   30000,
							Protocol:   v1.ProtocolTCP,
						},
						{
							Port:       80,
							TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: 8080},
							NodePort:   31000,
							Protocol:   v1.ProtocolUDP,
						},
					},
					HealthCheckNodePort: 32000,
				},
			},
			nodes: []*v1.Node{
				makeNode("a", "10.0.0.1"),
				makeNode("b", "10.0.0.2"),
			},
			want: &proxyConfigData{
				HealthCheckPort: 32000,
				ServicePorts: map[string]servicePort{
					"IPv4_80_TCP": servicePort{
						Listener: endpoint{Address: "0.0.0.0", Port: 80, Protocol: string(v1.ProtocolTCP)},
						Cluster:  []endpoint{{"10.0.0.1", 30000, string(v1.ProtocolTCP)}, {"10.0.0.2", 30000, string(v1.ProtocolTCP)}},
					},
					"IPv4_80_UDP": servicePort{
						Listener: endpoint{Address: "0.0.0.0", Port: 80, Protocol: string(v1.ProtocolUDP)},
						Cluster:  []endpoint{{"10.0.0.1", 31000, string(v1.ProtocolUDP)}, {"10.0.0.2", 31000, string(v1.ProtocolUDP)}},
					},
				},
			},
		},
		{
			name: "multiport service ipv6",
			service: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: v1.ServiceSpec{
					Type:                  v1.ServiceTypeLoadBalancer,
					ExternalTrafficPolicy: v1.ServiceExternalTrafficPolicyLocal,
					IPFamilies:            []v1.IPFamily{v1.IPv6Protocol},
					Ports: []v1.ServicePort{
						{
							Port:       80,
							TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: 8080},
							NodePort:   30000,
							Protocol:   v1.ProtocolTCP,
						},
						{
							Port:       443,
							TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: 8080},
							NodePort:   31000,
							Protocol:   v1.ProtocolTCP,
						},
					},
					HealthCheckNodePort: 32000,
				},
			},
			nodes: []*v1.Node{
				makeNode("a", "2001:db2::3"),
				makeNode("b", "2001:db2::4"),
			},
			want: &proxyConfigData{
				HealthCheckPort: 32000,
				ServicePorts: map[string]servicePort{
					"IPv6_80_TCP": servicePort{
						Listener: endpoint{Address: `"::"`, Port: 80, Protocol: string(v1.ProtocolTCP)},
						Cluster:  []endpoint{{"2001:db2::3", 30000, string(v1.ProtocolTCP)}, {"2001:db2::4", 30000, string(v1.ProtocolTCP)}},
					},
					"IPv6_443_TCP": servicePort{
						Listener: endpoint{Address: `"::"`, Port: 443, Protocol: string(v1.ProtocolTCP)},
						Cluster:  []endpoint{{"2001:db2::3", 31000, string(v1.ProtocolTCP)}, {"2001:db2::4", 31000, string(v1.ProtocolTCP)}},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := generateConfig(tt.service, tt.nodes); !reflect.DeepEqual(got, tt.want) {
				t.Logf("diff %+v", cmp.Diff(got, tt.want))
				t.Errorf("generateConfig() = %+v,\n want %+v", got, tt.want)
			}
		})
	}
}

func Test_proxyConfig(t *testing.T) {
	tests := []struct {
		name       string
		data       *proxyConfigData
		wantConfig string
	}{
		{
			name: "ipv4",
			data: &proxyConfigData{
				HealthCheckPort: 32764,
				ServicePorts: map[string]servicePort{
					"IPv4_80": servicePort{
						Listener: endpoint{Address: "0.0.0.0", Port: 80, Protocol: string(v1.ProtocolTCP)},
						Cluster:  []endpoint{{"192.168.8.2", 30497, string(v1.ProtocolTCP)}, {"192.168.8.3", 30497, string(v1.ProtocolTCP)}},
					},
					"IPv4_443": servicePort{
						Listener: endpoint{Address: "0.0.0.0", Port: 443, Protocol: string(v1.ProtocolTCP)},
						Cluster:  []endpoint{{"192.168.8.2", 31497, string(v1.ProtocolTCP)}, {"192.168.8.3", 31497, string(v1.ProtocolTCP)}},
					},
				},
			},
			wantConfig: `
admin:
  address:
    socket_address: { address: 127.0.0.1, port_value: 9901 }

static_resources:
  listeners:
  - name: listener_IPv4_443
    address:
      socket_address:
        address: 0.0.0.0
        port_value: 443
        protocol: TCP
    filter_chains:
      - filters:
        - name: envoy.filters.network.tcp_proxy
          typed_config:
            "@type": type.googleapis.com/envoy.extensions.filters.network.tcp_proxy.v3.TcpProxy
            stat_prefix: destination
            cluster: cluster_IPv4_443
  - name: listener_IPv4_80
    address:
      socket_address:
        address: 0.0.0.0
        port_value: 80
        protocol: TCP
    filter_chains:
      - filters:
        - name: envoy.filters.network.tcp_proxy
          typed_config:
            "@type": type.googleapis.com/envoy.extensions.filters.network.tcp_proxy.v3.TcpProxy
            stat_prefix: destination
            cluster: cluster_IPv4_80

  clusters:
  - name: cluster_IPv4_443
    connect_timeout: 5s
    type: STATIC
    lb_policy: RANDOM
    health_checks:
      - timeout: 5s
        interval: 3s
        unhealthy_threshold: 3
        healthy_threshold: 1
        always_log_health_check_failures: true
        always_log_health_check_success: true
        http_health_check:
          path: /healthz
    load_assignment:
      cluster_name: cluster_IPv4_443
      endpoints:
        - lb_endpoints:
          - endpoint:
              health_check_config:
                port_value: 32764
              address:
                socket_address:
                  address: 192.168.8.2
                  port_value: 31497
                  protocol: TCP
        - lb_endpoints:
          - endpoint:
              health_check_config:
                port_value: 32764
              address:
                socket_address:
                  address: 192.168.8.3
                  port_value: 31497
                  protocol: TCP
  - name: cluster_IPv4_80
    connect_timeout: 5s
    type: STATIC
    lb_policy: RANDOM
    health_checks:
      - timeout: 5s
        interval: 3s
        unhealthy_threshold: 3
        healthy_threshold: 1
        always_log_health_check_failures: true
        always_log_health_check_success: true
        http_health_check:
          path: /healthz
    load_assignment:
      cluster_name: cluster_IPv4_80
      endpoints:
        - lb_endpoints:
          - endpoint:
              health_check_config:
                port_value: 32764
              address:
                socket_address:
                  address: 192.168.8.2
                  port_value: 30497
                  protocol: TCP
        - lb_endpoints:
          - endpoint:
              health_check_config:
                port_value: 32764
              address:
                socket_address:
                  address: 192.168.8.3
                  port_value: 30497
                  protocol: TCP
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotConfig, err := proxyConfig(tt.data)
			if err != nil {
				t.Errorf("proxyConfig() error = %v", err)
				return
			}
			if gotConfig != tt.wantConfig {
				t.Errorf("proxyConfig() not expected\n%v", cmp.Diff(gotConfig, tt.wantConfig))
			}
		})
	}
}
