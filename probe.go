package main

func probe(
	a int,
	b int,
) int {
	return add(
		a,
		b,
		0,
	)
}

func add(
	a,
	b,
	c int,
) int {
	return a + b + c
}
