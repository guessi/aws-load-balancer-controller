package gatewayutils

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/ptr"
	elbv2gw "sigs.k8s.io/aws-load-balancer-controller/apis/gateway/v1beta1"
	"sigs.k8s.io/aws-load-balancer-controller/pkg/gateway/constants"
	"sigs.k8s.io/aws-load-balancer-controller/pkg/testutils"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
	"testing"
)

func Test_IsGatewayManagedByLBController(t *testing.T) {
	type args struct {
		gateway      *gwv1.Gateway
		gwClass      *gwv1.GatewayClass
		gwController string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "gateway with valid NLB controller",
			args: args{
				gateway: &gwv1.Gateway{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-nlb-gw",
					},
					Spec: gwv1.GatewaySpec{
						GatewayClassName: "nlb-class",
					},
				},
				gwClass: &gwv1.GatewayClass{
					ObjectMeta: metav1.ObjectMeta{
						Name: "nlb-class",
					},
					Spec: gwv1.GatewayClassSpec{
						ControllerName: constants.NLBGatewayController,
					},
				},
				gwController: constants.NLBGatewayController,
			},
			want: true,
		},
		{
			name: "gateway with valid ALB controller",
			args: args{
				gateway: &gwv1.Gateway{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-alb-gw",
					},
					Spec: gwv1.GatewaySpec{
						GatewayClassName: "alb-class",
					},
				},
				gwClass: &gwv1.GatewayClass{
					ObjectMeta: metav1.ObjectMeta{
						Name: "alb-class",
					},
					Spec: gwv1.GatewayClassSpec{
						ControllerName: constants.ALBGatewayController,
					},
				},
				gwController: constants.ALBGatewayController,
			},
			want: true,
		},
		{
			name: "gateway with invalid controller",
			args: args{
				gateway: &gwv1.Gateway{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-invalid-gw",
					},
					Spec: gwv1.GatewaySpec{
						GatewayClassName: "invalid-class",
					},
				},
				gwClass: &gwv1.GatewayClass{
					ObjectMeta: metav1.ObjectMeta{
						Name: "invalid-class",
					},
					Spec: gwv1.GatewayClassSpec{
						ControllerName: "some.other.controller",
					},
				},
				gwController: constants.ALBGatewayController,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k8sClient := testutils.GenerateTestClient()
			k8sClient.Create(context.Background(), tt.args.gwClass)
			k8sClient.Create(context.Background(), tt.args.gateway)
			got := IsGatewayManagedByLBController(context.Background(), k8sClient, tt.args.gateway, tt.args.gwController)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_GetGatewayClassesManagedByLBController(t *testing.T) {
	type args struct {
		gwClasses     []*gwv1.GatewayClass
		gwControllers sets.Set[string]
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "multiple gateway classes for NLB Gateway controller",
			args: args{
				gwClasses: []*gwv1.GatewayClass{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "nlb-class-1",
						},
						Spec: gwv1.GatewayClassSpec{
							ControllerName: constants.NLBGatewayController,
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "nlb-class-2",
						},
						Spec: gwv1.GatewayClassSpec{
							ControllerName: constants.NLBGatewayController,
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "alb-class",
						},
						Spec: gwv1.GatewayClassSpec{
							ControllerName: constants.ALBGatewayController,
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "invalid-class",
						},
						Spec: gwv1.GatewayClassSpec{
							ControllerName: "some.other.controller",
						},
					},
				},
				gwControllers: sets.New(constants.NLBGatewayController),
			},
			want: 2,
		},
		{
			name: "multiple gateway classes for ALB Gateway controller",
			args: args{
				gwClasses: []*gwv1.GatewayClass{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "nlb-class-1",
						},
						Spec: gwv1.GatewayClassSpec{
							ControllerName: constants.NLBGatewayController,
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "alb-class-1",
						},
						Spec: gwv1.GatewayClassSpec{
							ControllerName: constants.ALBGatewayController,
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "alb-class-2",
						},
						Spec: gwv1.GatewayClassSpec{
							ControllerName: constants.ALBGatewayController,
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "invalid-class",
						},
						Spec: gwv1.GatewayClassSpec{
							ControllerName: "some.other.controller",
						},
					},
				},
				gwControllers: sets.New(constants.ALBGatewayController),
			},
			want: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k8sClient := testutils.GenerateTestClient()

			for _, gwClass := range tt.args.gwClasses {
				k8sClient.Create(context.Background(), gwClass)
			}

			got, err := GetGatewayClassesManagedByLBController(context.Background(), k8sClient, tt.args.gwControllers)
			assert.Equal(t, tt.want, len(got))
			assert.NoError(t, err)
		})
	}
}

func Test_GetImpactedGatewaysFromParentRefs(t *testing.T) {
	type args struct {
		parentRefs                        []gwv1.ParentReference
		originalParentRefsFromRouteStatus []gwv1.RouteParentStatus
		resourceNS                        string
		gateways                          []*gwv1.Gateway
		gatewayClasses                    []*gwv1.GatewayClass
		gwController                      string
	}
	tests := []struct {
		name    string
		args    args
		want    []types.NamespacedName
		wantErr error
	}{
		{
			name: "valid parent refs with managed gateways",
			args: args{
				parentRefs: []gwv1.ParentReference{
					{
						Name:      "test-gw",
						Namespace: (*gwv1.Namespace)(ptr.To("test-ns")),
					},
				},
				resourceNS: "test-ns",
				gateways: []*gwv1.Gateway{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-gw",
							Namespace: "test-ns",
						},
						Spec: gwv1.GatewaySpec{
							GatewayClassName: "nlb-class",
						},
					},
				},
				gatewayClasses: []*gwv1.GatewayClass{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "nlb-class",
						},
						Spec: gwv1.GatewayClassSpec{
							ControllerName: constants.NLBGatewayController,
						},
					},
				},
				gwController: constants.NLBGatewayController,
			},
			want: []types.NamespacedName{
				{
					Namespace: "test-ns",
					Name:      "test-gw",
				},
			},
			wantErr: nil,
		},
		{
			name: "valid parent refs with managed gateways and originalParentRefsFromRouteStatus",
			args: args{
				parentRefs: []gwv1.ParentReference{
					{
						Name:      "test-gw",
						Namespace: (*gwv1.Namespace)(ptr.To("test-ns")),
					},
				},
				resourceNS: "test-ns",
				originalParentRefsFromRouteStatus: []gwv1.RouteParentStatus{
					{
						ParentRef: gwv1.ParentReference{
							Name:      "test-gw-1",
							Namespace: (*gwv1.Namespace)(ptr.To("test-ns")),
						},
					},
				},
				gateways: []*gwv1.Gateway{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-gw",
							Namespace: "test-ns",
						},
						Spec: gwv1.GatewaySpec{
							GatewayClassName: "nlb-class",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-gw-1",
							Namespace: "test-ns",
						},
						Spec: gwv1.GatewaySpec{
							GatewayClassName: "nlb-class",
						},
					},
				},
				gatewayClasses: []*gwv1.GatewayClass{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "nlb-class",
						},
						Spec: gwv1.GatewayClassSpec{
							ControllerName: constants.NLBGatewayController,
						},
					},
				},
				gwController: constants.NLBGatewayController,
			},
			want: []types.NamespacedName{
				{
					Namespace: "test-ns",
					Name:      "test-gw",
				},
				{
					Namespace: "test-ns",
					Name:      "test-gw-1",
				},
			},
			wantErr: nil,
		},
		{
			name: "valid parent refs with unmanaged gateways",
			args: args{
				parentRefs: []gwv1.ParentReference{
					{
						Name:      "test-gw",
						Namespace: (*gwv1.Namespace)(ptr.To("test-ns")),
					},
				},
				resourceNS: "test-ns",
				gateways: []*gwv1.Gateway{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-gw",
							Namespace: "test-ns",
						},
						Spec: gwv1.GatewaySpec{
							GatewayClassName: "alb-class",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-unmanaged-gw",
							Namespace: "test-ns",
						},
						Spec: gwv1.GatewaySpec{
							GatewayClassName: "unmanaged-class",
						},
					},
				},
				gatewayClasses: []*gwv1.GatewayClass{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "alb-class",
						},
						Spec: gwv1.GatewayClassSpec{
							ControllerName: constants.ALBGatewayController,
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "unmanaged-class",
						},
						Spec: gwv1.GatewayClassSpec{
							ControllerName: "some.other.controller",
						},
					},
				},
				gwController: constants.ALBGatewayController,
			},
			want: []types.NamespacedName{
				{
					Namespace: "test-ns",
					Name:      "test-gw",
				},
			},
			wantErr: nil,
		},
		{
			name: "valid parent refs with unmanaged and unimpacted gateways",
			args: args{
				parentRefs: []gwv1.ParentReference{
					{
						Name:      "test-gw",
						Namespace: (*gwv1.Namespace)(ptr.To("test-ns")),
					},
				},
				resourceNS: "test-ns",
				gateways: []*gwv1.Gateway{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-gw",
							Namespace: "test-ns",
						},
						Spec: gwv1.GatewaySpec{
							GatewayClassName: "alb-class",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-unimpacted-gw",
							Namespace: "another-ns",
						},
						Spec: gwv1.GatewaySpec{
							GatewayClassName: "alb-class",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-unmanaged-gw",
							Namespace: "test-ns",
						},
						Spec: gwv1.GatewaySpec{
							GatewayClassName: "unmanaged-class",
						},
					},
				},
				gatewayClasses: []*gwv1.GatewayClass{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "alb-class",
						},
						Spec: gwv1.GatewayClassSpec{
							ControllerName: constants.ALBGatewayController,
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "unmanaged-class",
						},
						Spec: gwv1.GatewayClassSpec{
							ControllerName: "some.other.controller",
						},
					},
				},
				gwController: constants.ALBGatewayController,
			},
			want: []types.NamespacedName{
				{
					Namespace: "test-ns",
					Name:      "test-gw",
				},
			},
			wantErr: nil,
		},
		{
			name: "valid parent refs with managed gateways and unknown gateways",
			args: args{
				parentRefs: []gwv1.ParentReference{
					{
						Name:      "test-gw",
						Namespace: (*gwv1.Namespace)(ptr.To("test-ns")),
					},
					{
						Name:      "unknown-gw",
						Namespace: (*gwv1.Namespace)(ptr.To("test-ns")),
					},
				},
				resourceNS: "test-ns",
				gateways: []*gwv1.Gateway{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-gw",
							Namespace: "test-ns",
						},
						Spec: gwv1.GatewaySpec{
							GatewayClassName: "nlb-class",
						},
					},
				},
				gatewayClasses: []*gwv1.GatewayClass{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "nlb-class",
						},
						Spec: gwv1.GatewayClassSpec{
							ControllerName: constants.NLBGatewayController,
						},
					},
				},
				gwController: constants.NLBGatewayController,
			},
			want: []types.NamespacedName{
				{
					Namespace: "test-ns",
					Name:      "test-gw",
				},
			},
			wantErr: fmt.Errorf("failed to list gateways, [%s]", types.NamespacedName{Namespace: "test-ns", Name: "unknown-gw"}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k8sClient := testutils.GenerateTestClient()

			for _, gw := range tt.args.gateways {
				k8sClient.Create(context.Background(), gw)
			}
			for _, gwClass := range tt.args.gatewayClasses {
				k8sClient.Create(context.Background(), gwClass)
			}

			got, err := GetImpactedGatewaysFromParentRefs(context.Background(), k8sClient, tt.args.parentRefs, tt.args.originalParentRefsFromRouteStatus, tt.args.resourceNS, tt.args.gwController)

			assert.Equal(t, err, tt.wantErr)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_GetImpactedGatewayClassesFromLbConfig(t *testing.T) {
	defaultNamespace := gwv1.Namespace("default")
	anotherNamespace := gwv1.Namespace("another-namespace")
	type args struct {
		lbConfig      *elbv2gw.LoadBalancerConfiguration
		gwClasses     []*gwv1.GatewayClass
		gwControllers sets.Set[string]
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "matching and non-matching lb config reference for ALB Gateway Controller",
			args: args{
				lbConfig: &elbv2gw.LoadBalancerConfiguration{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-config",
						Namespace: "default",
					},
				},
				gwClasses: []*gwv1.GatewayClass{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "test-class-1",
						},
						Spec: gwv1.GatewayClassSpec{
							ControllerName: constants.NLBGatewayController,
							ParametersRef: &gwv1.ParametersReference{
								Kind:      "LoadBalancerConfiguration",
								Name:      "test-config",
								Namespace: &defaultNamespace,
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "test-class-2",
						},
						Spec: gwv1.GatewayClassSpec{
							ControllerName: constants.ALBGatewayController,
							ParametersRef: &gwv1.ParametersReference{
								Kind:      "LoadBalancerConfiguration",
								Name:      "test-config",
								Namespace: &defaultNamespace,
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "test-class",
						},
						Spec: gwv1.GatewayClassSpec{
							ControllerName: constants.NLBGatewayController,
							ParametersRef: &gwv1.ParametersReference{
								Kind:      "LoadBalancerConfiguration",
								Name:      "test-config",
								Namespace: &anotherNamespace,
							},
						},
					},
				},
				gwControllers: sets.New(constants.ALBGatewayController),
			},
			want: 1,
		},
		{
			name: "matching and non-matching lb config reference for NLB Gateway Controller",
			args: args{
				lbConfig: &elbv2gw.LoadBalancerConfiguration{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-config",
						Namespace: "default",
					},
				},
				gwClasses: []*gwv1.GatewayClass{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "test-class-1",
						},
						Spec: gwv1.GatewayClassSpec{
							ControllerName: constants.NLBGatewayController,
							ParametersRef: &gwv1.ParametersReference{
								Kind:      "LoadBalancerConfiguration",
								Name:      "test-config",
								Namespace: &defaultNamespace,
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "test-class-2",
						},
						Spec: gwv1.GatewayClassSpec{
							ControllerName: constants.ALBGatewayController,
							ParametersRef: &gwv1.ParametersReference{
								Kind:      "LoadBalancerConfiguration",
								Name:      "test-config",
								Namespace: &defaultNamespace,
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "test-class",
						},
						Spec: gwv1.GatewayClassSpec{
							ControllerName: constants.NLBGatewayController,
							ParametersRef: &gwv1.ParametersReference{
								Kind:      "LoadBalancerConfiguration",
								Name:      "test-config",
								Namespace: &anotherNamespace,
							},
						},
					},
				},
				gwControllers: sets.New(constants.NLBGatewayController),
			},
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k8sClient := testutils.GenerateTestClient()
			for _, gwClass := range tt.args.gwClasses {
				k8sClient.Create(context.Background(), gwClass)
			}

			got, err := GetImpactedGatewayClassesFromLbConfig(context.Background(), k8sClient, tt.args.lbConfig, tt.args.gwControllers)
			assert.Equal(t, tt.want, len(got))
			assert.NoError(t, err)
		})
	}
}

func Test_GetImpactedGatewaysFromLbConfig(t *testing.T) {
	type args struct {
		lbConfig       *elbv2gw.LoadBalancerConfiguration
		gateways       []*gwv1.Gateway
		gatewayClasses []*gwv1.GatewayClass
		gwController   string
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "matching and unmanaged lb config reference for ALB Gateway Controller",
			args: args{
				lbConfig: &elbv2gw.LoadBalancerConfiguration{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-config",
					},
				},
				gateways: []*gwv1.Gateway{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "test-managed-gw",
						},
						Spec: gwv1.GatewaySpec{
							GatewayClassName: "test-managed-class",
							Infrastructure: &gwv1.GatewayInfrastructure{
								ParametersRef: &gwv1.LocalParametersReference{
									Kind: "LoadBalancerConfiguration",
									Name: "test-config",
								},
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "test-unmatched-gw",
						},
						Spec: gwv1.GatewaySpec{
							GatewayClassName: "test-managed-class",
							Infrastructure: &gwv1.GatewayInfrastructure{
								ParametersRef: nil,
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "test-unmanaged-gw",
						},
						Spec: gwv1.GatewaySpec{
							GatewayClassName: "test-unmanaged-class",
							Infrastructure: &gwv1.GatewayInfrastructure{
								ParametersRef: &gwv1.LocalParametersReference{
									Kind: "LoadBalancerConfiguration",
									Name: "test-config",
								},
							},
						},
					},
				},
				gatewayClasses: []*gwv1.GatewayClass{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "test-managed-class",
						},
						Spec: gwv1.GatewayClassSpec{
							ControllerName: constants.ALBGatewayController,
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "test-unmanaged-class",
						},
						Spec: gwv1.GatewayClassSpec{
							ControllerName: constants.NLBGatewayController,
						},
					},
				},
				gwController: constants.ALBGatewayController,
			},
			want: 1,
		},
		{
			name: "matching and unmanaged lb config reference for NLB Gateway Controller",
			args: args{
				lbConfig: &elbv2gw.LoadBalancerConfiguration{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-config",
					},
				},
				gateways: []*gwv1.Gateway{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-managed-gw",
							Namespace: "other namespace",
						},
						Spec: gwv1.GatewaySpec{
							GatewayClassName: "test-managed-class",
							Infrastructure: &gwv1.GatewayInfrastructure{
								ParametersRef: &gwv1.LocalParametersReference{
									Kind: "LoadBalancerConfiguration",
									Name: "test-config",
								},
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "test-managed-gw",
						},
						Spec: gwv1.GatewaySpec{
							GatewayClassName: "test-managed-class",
							Infrastructure: &gwv1.GatewayInfrastructure{
								ParametersRef: &gwv1.LocalParametersReference{
									Kind: "LoadBalancerConfiguration",
									Name: "test-config",
								},
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "test-unmatched-gw",
						},
						Spec: gwv1.GatewaySpec{
							GatewayClassName: "test-managed-class",
							Infrastructure: &gwv1.GatewayInfrastructure{
								ParametersRef: nil,
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "test-unmanaged-gw",
						},
						Spec: gwv1.GatewaySpec{
							GatewayClassName: "test-unmanaged-class",
							Infrastructure: &gwv1.GatewayInfrastructure{
								ParametersRef: &gwv1.LocalParametersReference{
									Kind: "LoadBalancerConfiguration",
									Name: "test-config",
								},
							},
						},
					},
				},
				gatewayClasses: []*gwv1.GatewayClass{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "test-managed-class",
						},
						Spec: gwv1.GatewayClassSpec{
							ControllerName: constants.NLBGatewayController,
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "test-unmanaged-class",
						},
						Spec: gwv1.GatewayClassSpec{
							ControllerName: constants.ALBGatewayController,
						},
					},
				},
				gwController: constants.NLBGatewayController,
			},
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k8sClient := testutils.GenerateTestClient()
			for _, gwClass := range tt.args.gatewayClasses {
				k8sClient.Create(context.Background(), gwClass)
			}
			for _, gw := range tt.args.gateways {
				k8sClient.Create(context.Background(), gw)
			}
			got, err := GetImpactedGatewaysFromLbConfig(context.Background(), k8sClient, tt.args.lbConfig, tt.args.gwController)
			assert.Equal(t, tt.want, len(got))
			assert.NoError(t, err)
		})
	}
}

func Test_GetGatewaysManagedByGatewayClass(t *testing.T) {
	type args struct {
		gateways  []*gwv1.Gateway
		gwClasses []*gwv1.GatewayClass
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "multiple gateways for nlb gw classes",
			args: args{
				gateways: []*gwv1.Gateway{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-gw",
							Namespace: "test-ns",
						},
						Spec: gwv1.GatewaySpec{
							GatewayClassName: "nlb-class",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-gw-1",
							Namespace: "test-ns",
						},
						Spec: gwv1.GatewaySpec{
							GatewayClassName: "nlb-class",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-alb-gw-1",
							Namespace: "test-ns",
						},
						Spec: gwv1.GatewaySpec{
							GatewayClassName: "alb-class",
						},
					},
				},
				gwClasses: []*gwv1.GatewayClass{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "nlb-class",
						},
						Spec: gwv1.GatewayClassSpec{
							ControllerName: constants.NLBGatewayController,
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "alb-class",
						},
						Spec: gwv1.GatewayClassSpec{
							ControllerName: constants.ALBGatewayController,
						},
					},
				},
			},
			want: 2,
		},
		{
			name: "multiple gateways for alb gw classes",
			args: args{
				gateways: []*gwv1.Gateway{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-gw",
							Namespace: "test-ns",
						},
						Spec: gwv1.GatewaySpec{
							GatewayClassName: "nlb-class",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-gw-1",
							Namespace: "test-ns",
						},
						Spec: gwv1.GatewaySpec{
							GatewayClassName: "nlb-class",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-alb-gw-1",
							Namespace: "test-ns",
						},
						Spec: gwv1.GatewaySpec{
							GatewayClassName: "alb-class",
						},
					},
				},
				gwClasses: []*gwv1.GatewayClass{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "alb-class",
						},
						Spec: gwv1.GatewayClassSpec{
							ControllerName: constants.ALBGatewayController,
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "nlb-class",
						},
						Spec: gwv1.GatewayClassSpec{
							ControllerName: constants.NLBGatewayController,
						},
					},
				},
			},
			want: 1,
		},
		{
			name: "no valid gateways for nlb gw classes",
			args: args{
				gateways: []*gwv1.Gateway{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-alb-gw-1",
							Namespace: "test-ns",
						},
						Spec: gwv1.GatewaySpec{
							GatewayClassName: "alb-class",
						},
					},
				},
				gwClasses: []*gwv1.GatewayClass{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "nlb-class",
						},
						Spec: gwv1.GatewayClassSpec{
							ControllerName: constants.NLBGatewayController,
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "alb-class",
						},
						Spec: gwv1.GatewayClassSpec{
							ControllerName: constants.ALBGatewayController,
						},
					},
				},
			},
			want: 0,
		},
		{
			name: "no valid gateways for alb gw classes",
			args: args{
				gateways: []*gwv1.Gateway{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-gw",
							Namespace: "test-ns",
						},
						Spec: gwv1.GatewaySpec{
							GatewayClassName: "nlb-class",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-gw-1",
							Namespace: "test-ns",
						},
						Spec: gwv1.GatewaySpec{
							GatewayClassName: "nlb-class",
						},
					},
				},
				gwClasses: []*gwv1.GatewayClass{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "alb-class",
						},
						Spec: gwv1.GatewayClassSpec{
							ControllerName: constants.ALBGatewayController,
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "nlb-class",
						},
						Spec: gwv1.GatewayClassSpec{
							ControllerName: constants.NLBGatewayController,
						},
					},
				},
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k8sClient := testutils.GenerateTestClient()
			for _, gwClass := range tt.args.gwClasses {
				k8sClient.Create(context.Background(), gwClass)
			}
			for _, gw := range tt.args.gateways {
				k8sClient.Create(context.Background(), gw)
			}

			got, err := GetGatewaysManagedByGatewayClass(context.Background(), k8sClient, tt.args.gwClasses[0])
			assert.Equal(t, tt.want, len(got))
			assert.NoError(t, err)
		})
	}
}

func TestRemoveDuplicateParentRefs(t *testing.T) {
	namespace := "test-namespace"
	tests := []struct {
		name              string
		parentRefs        []gwv1.ParentReference
		resourceNamespace string
		want              []gwv1.ParentReference
	}{
		{
			name: "no duplicates",
			parentRefs: []gwv1.ParentReference{
				{Name: "gateway1"},
				{Name: "gateway2"},
			},
			resourceNamespace: namespace,
			want: []gwv1.ParentReference{
				{Name: "gateway1"},
				{Name: "gateway2"},
			},
		},
		{
			name: "one duplicate",
			parentRefs: []gwv1.ParentReference{
				{Name: "gateway1"},
				{Name: "gateway1"},
				{Name: "gateway2"},
			},
			resourceNamespace: namespace,
			want: []gwv1.ParentReference{
				{Name: "gateway1"},
				{Name: "gateway2"},
			},
		},
		{
			name: "multiple duplicates",
			parentRefs: []gwv1.ParentReference{
				{Name: "gateway1"},
				{Name: "gateway2"},
				{Name: "gateway1"},
				{Name: "gateway2"},
				{Name: "gateway3"},
				{Name: "gateway1"},
			},
			resourceNamespace: namespace,
			want: []gwv1.ParentReference{
				{Name: "gateway1"},
				{Name: "gateway2"},
				{Name: "gateway3"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := removeDuplicateParentRefs(tt.parentRefs, tt.resourceNamespace)

			assert.Equal(t, len(tt.want), len(got))

			actual := make(map[string]bool)
			for _, ref := range got {
				actual[string(ref.Name)] = true
			}

			expected := make(map[string]bool)
			for _, ref := range tt.want {
				expected[string(ref.Name)] = true
			}

			assert.Equal(t, expected, actual)
		})
	}
}
