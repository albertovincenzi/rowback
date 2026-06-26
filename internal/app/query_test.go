package app

import "testing"

func TestParseDefaultQuery(t *testing.T) {
	q, err := ParseQuery(defaultQuery)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if q.Table != "orders" {
		t.Fatalf("table = %q, want orders", q.Table)
	}
	if len(q.Where) != 2 {
		t.Fatalf("got %d conditions, want 2: %+v", len(q.Where), q.Where)
	}

	// cond 0: customer_id IN (SELECT id FROM customers WHERE region_id = 42)
	c0 := q.Where[0]
	if c0.Col != "customer_id" || c0.Sub == nil {
		t.Fatalf("cond0 wrong: %+v", c0)
	}
	if c0.Sub.Table != "customers" || c0.Sub.Col != "id" {
		t.Fatalf("subquery wrong: %+v", c0.Sub)
	}
	if len(c0.Sub.Where) != 1 || c0.Sub.Where[0].Col != "region_id" ||
		c0.Sub.Where[0].Op != "=" || c0.Sub.Where[0].Values[0] != "42" {
		t.Fatalf("subquery where wrong: %+v", c0.Sub.Where)
	}

	// cond 1: order_no IN ('A-1001','A-1002')
	c1 := q.Where[1]
	if c1.Col != "order_no" || c1.op() != "in" || len(c1.Values) != 2 {
		t.Fatalf("cond1 wrong: %+v", c1)
	}
	want := map[string]bool{"A-1001": true, "A-1002": true}
	for _, v := range c1.Values {
		if !want[v] {
			t.Fatalf("unexpected order_no %q", v)
		}
	}
}

func TestParseRejectsNoWhere(t *testing.T) {
	if _, err := ParseQuery("DELETE FROM orders"); err == nil {
		t.Fatal("expected error for missing WHERE")
	}
}

func TestSplitTopLevelAnd(t *testing.T) {
	// AND inside the subquery parens must not split the top level.
	parts := splitTopLevelAnd("a IN (SELECT x FROM y WHERE p = 1 AND q = 2) AND b = 3")
	if len(parts) != 2 {
		t.Fatalf("got %d parts, want 2: %q", len(parts), parts)
	}
}
