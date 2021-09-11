package lineage

import (
	"fmt"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	unstructuredv1 "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/duration"
	"k8s.io/client-go/util/jsonpath"
)

const (
	cellUnknown = "<unknown>"
	cellUnset   = "<none>"
)

var (
	objectColumnDefinitions = []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Status", Type: "string", Description: "The condition Ready status of the object."},
		{Name: "Reason", Type: "string", Description: "The condition Ready reason of the object."},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
	}
)

// objectColumns holds columns for all kinds of Kubernetes objects
type objectColumns struct {
	Name   string
	Status string
	Reason string
	Age    string
}

// TODO: Sort dependents before printing
// TODO: Refactor this to remove duplication
func printNodeMap(nodeMap NodeMap, uid types.UID, prefix string, showGroup bool) ([]metav1.TableRow, error) {
	// Track every object kind in the node map & the groups that they belong to.
	// When printing an object & if there exists another object in the node map
	// that has the same kind but belongs to a different group (eg. "services.v1"
	// vs "services.v1.serving.knative.dev"), we prepend the object's name with
	// its GroupKind instead of its Kind to clearly indicate which resource type
	// it belongs to.
	kindToGroupSetMap := map[string](map[string]struct{}){}
	for _, node := range nodeMap {
		gvk := node.GroupVersionKind()
		if _, ok := kindToGroupSetMap[gvk.Kind]; !ok {
			kindToGroupSetMap[gvk.Kind] = map[string]struct{}{}
		}
		kindToGroupSetMap[gvk.Kind][gvk.Group] = struct{}{}
	}

	var rows []metav1.TableRow
	node := nodeMap[uid]

	if len(prefix) == 0 {
		showGroup := len(kindToGroupSetMap[node.GroupVersionKind().Kind]) > 1 || showGroup
		columns := getObjectColumns(*node.Unstructured, showGroup)
		row := metav1.TableRow{
			Object: runtime.RawExtension{
				Object: node.DeepCopyObject(),
			},
			Cells: []interface{}{
				columns.Name,
				columns.Status,
				columns.Reason,
				columns.Age,
			},
		}
		rows = append(rows, row)
	}

	for i, childUID := range node.Dependents {
		child := nodeMap[childUID]

		// Compute prefix
		var rowPrefix, childPrefix string
		if i != len(node.Dependents)-1 {
			rowPrefix, childPrefix = prefix+"├── ", prefix+"│   "
		} else {
			rowPrefix, childPrefix = prefix+"└── ", prefix+"    "
		}

		showGroup := len(kindToGroupSetMap[child.GroupVersionKind().Kind]) > 1 || showGroup
		columns := getObjectColumns(*child.Unstructured, showGroup)
		row := metav1.TableRow{
			Object: runtime.RawExtension{
				Object: child.DeepCopyObject(),
			},
			Cells: []interface{}{
				rowPrefix + columns.Name,
				columns.Status,
				columns.Reason,
				columns.Age,
			},
		}
		rows = append(rows, row)

		childRows, err := printNodeMap(nodeMap, childUID, childPrefix, showGroup)
		if err != nil {
			return nil, err
		}
		rows = append(rows, childRows...)
	}

	return rows, nil
}

func getNestedString(u unstructuredv1.Unstructured, name, jsonPath string) (string, error) {
	jp := jsonpath.New(name).AllowMissingKeys(true)
	if err := jp.Parse(jsonPath); err != nil {
		return "", err
	}

	data := u.UnstructuredContent()
	values, err := jp.FindResults(data)
	if err != nil {
		return "", err
	}
	strValues := []string{}
	for arrIx := range values {
		for valIx := range values[arrIx] {
			strValues = append(strValues, fmt.Sprintf("%v", values[arrIx][valIx].Interface()))
		}
	}
	str := strings.Join(strValues, ",")

	return str, nil
}

func getObjectColumns(u unstructuredv1.Unstructured, showGroup bool) *objectColumns {
	k, gk, name := u.GetKind(), u.GroupVersionKind().GroupKind(), u.GetName()
	if showGroup {
		name = fmt.Sprintf("%s/%s", gk, name)
	} else {
		name = fmt.Sprintf("%s/%s", k, name)
	}
	status, _ := getNestedString(u, "condition-ready-status", "{.status.conditions[?(@.type==\"Ready\")].status}")
	if len(status) == 0 {
		status = cellUnset
	}
	reason, _ := getNestedString(u, "condition-ready-reason", "{.status.conditions[?(@.type==\"Ready\")].reason}")
	if len(reason) == 0 {
		reason = cellUnset
	}

	return &objectColumns{
		Name:   name,
		Status: status,
		Reason: reason,
		Age:    translateTimestampSince(u.GetCreationTimestamp()),
	}
}

// translateTimestampSince returns the elapsed time since timestamp in
// human-readable approximation.
func translateTimestampSince(timestamp metav1.Time) string {
	if timestamp.IsZero() {
		return cellUnknown
	}

	return duration.HumanDuration(time.Since(timestamp.Time))
}
