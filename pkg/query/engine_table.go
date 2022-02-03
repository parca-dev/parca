package query

//
//import "testing"
//
//// This test models a query whose result is the available values for the "sample_type" column.
//func TestQuerySampleType(t *testing.T) {
//	engine := NewQueryEngine(table) // table is a "generic" source of arrow frames.
//
//	_, err := engine.Aggregate(
//		Unique(Column("sample_type")),
//	)
//	if err != nil {
//		t.Fatal(err)
//	}
//}
//
//// This test models a query whose result is the available values for the "sample_unit" column given a "sample_type" value.
//func TestQuerySampleUnit(t *testing.T) {
//	engine := NewQueryEngine(table) // table is a "generic" source of arrow frames.
//
//	_, err := engine.Filter(
//		Column("sample_type").Equal(StringScalar("cpu")),
//	).Aggregate(
//		Unique(Column("sample_unit")),
//	)
//	if err != nil {
//		t.Fatal(err)
//	}
//}
//
//// This test models a query whose result is all available label names.
//func TestQueryLabelNames(t *testing.T) {
//	engine := NewQueryEngine(table) // table is a "generic" source of arrow frames.
//
//	_, err := engine.Aggregate(
//		Unique(DynamicColumn("labels").Names()),
//	)
//	if err != nil {
//		t.Fatal(err)
//	}
//}
//
//// This test models a query whose result is all available label names.
//func TestQueryLabelValues(t *testing.T) {
//	engine := NewQueryEngine(table) // table is a "generic" source of arrow frames.
//
//	_, err := engine.Aggregate(
//		Unique(DynamicColumn("labels").Column("namespace")),
//	)
//	if err != nil {
//		t.Fatal(err)
//	}
//}
//
//// This test models a query whose result is a flat profile.
//func TestQueryFlatProfile(t *testing.T) {
//	engine := NewTableQueryEngine(table)
//
//	_, err := engine.Filter( // Parameters of Filter are a variadic list of Expressions.
//		Column("sample_type").Equal(StringScalar("cpu")),
//		Column("sample_unit").Equal(StringScalar("samples_count")),
//		DynamicColumn("labels").Column("namespace").Equal(StringScalar("default")),
//		Column("timestamp").GreaterThan(Int64Scalar(0)),
//		Column("timestamp").LessThan(Int64Scalar(0)),
//	).AggregateBy(
//		Sum(Column("value")), Column("stacktrace"),
//	)
//	if err != nil {
//		t.Fatal(err)
//	}
//}
//
//// This test models a query whose result is a flat profile where stacktraces
//// are filtered by containing a .
//func TestQueryFlatProfileFilterByStacktraceLocation(t *testing.T) {
//	engine := NewTableQueryEngine(table)
//
//	_, err := engine.Filter( // Parameters of Filter are a variadic list of Expressions.
//		Column("sample_type").Equal(StringScalar("cpu")),
//		Column("sample_unit").Equal(StringScalar("samples_count")),
//		DynamicColumn("labels").Column("namespace").Equal(StringScalar("default")),
//		Column("stacktrace").ContainsOneOf(UUIDScalar(0)),
//		Column("timestamp").GreaterThan(Int64Scalar(0)),
//		Column("timestamp").LessThan(Int64Scalar(0)),
//	).AggregateBy(
//		Sum(Column("value")), Column("stacktrace"),
//	)
//	if err != nil {
//		t.Fatal(err)
//	}
//}
//
//// This test models a query whose result is a flat profile where stacktraces
//// are filtered with a starting sublist.
//func TestQueryFlatProfileFilterByStacktrace(t *testing.T) {
//	engine := NewTableQueryEngine(table)
//
//	_, err := engine.Filter( // Parameters of Filter are a variadic list of Expressions.
//		Column("sample_type").Equal(StringScalar("cpu")),
//		Column("sample_unit").Equal(StringScalar("samples_count")),
//		DynamicColumn("labels").Column("namespace").Equal(StringScalar("default")),
//		Column("stacktrace").StartsWithList(UUIDScalar(0), UUIDScalar(1)),
//		Column("timestamp").GreaterThan(Int64Scalar(0)),
//		Column("timestamp").LessThan(Int64Scalar(0)),
//	).AggregateBy(
//		Sum(Column("value")), Column("stacktrace"),
//	)
//	if err != nil {
//		t.Fatal(err)
//	}
//}
//
//// This test models a query whose result is values aggregated by labels. In terms of profiling data this would mean the total CPU a process used at a point in time.
//func TestQueryTotalUsage(t *testing.T) {
//	engine := NewTableQueryEngine(table)
//
//	_, err := engine.Filter( // Parameters of Filter are a variadic list of Expressions.
//		Column("sample_type").Equal(StringScalar("cpu")),
//		Column("sample_unit").Equal(StringScalar("samples_count")),
//		DynamicColumn("labels").Column("namespace").Equal(StringScalar("default")),
//		Column("timestamp").GreaterThan(Int64Scalar(0)),
//		Column("timestamp").LessThan(Int64Scalar(0)),
//	).AggregateBy(
//		Sum(Column("value")), DynamicColumn("labels"), Column("timestamp"),
//	)
//	if err != nil {
//		t.Fatal(err)
//	}
//}
