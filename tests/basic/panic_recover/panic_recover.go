package main

import "fmt"

func a() {
	fmt.Printf("a starts\n")

	if err := recover(); err != nil {
		fmt.Printf("a recovers: %v\n", err)
	} else {
		fmt.Printf("a does not recover\n")
	}
	fmt.Println("a panics")
	panic(fmt.Errorf("a panicked"))

	fmt.Printf("b ends normally\n")
}

func b() {
	fmt.Printf("b starts\n")

	a()
	if err := recover(); err != nil {
		fmt.Println("b recovers: %v\n", err)
	} else {
		fmt.Printf("b does not recover\n")
	}
	fmt.Println("b panics")
	panic(fmt.Errorf("b panicked"))

	fmt.Printf("b ends normally\n")
}

func c() {
	fmt.Printf("c starts\n")
	fmt.Printf("c ends normally\n")
}

func d() {
	fmt.Printf("d starts\n")

	defer func() {
		fmt.Printf("d_closure_1 starts\n")

		defer func() {
			fmt.Printf("d_closure_1_closure starts\n")

			err := recover()
			if err != nil {
				fmt.Printf("d_closure_1_closure recovers: %v\n", err)
			} else {
				fmt.Printf("d_closure_1_closure does not recover\n")
			}

			fmt.Printf("d_closure_1_closure ends normally\n")
		}()

		fmt.Printf("d_closure_1 ends normally\n")
	}()
	defer func() {
		fmt.Printf("d_closure_2 starts\n")

		err := recover()
		if err != nil {
			fmt.Printf("d_closure_2 recovers: %v\n", err)
		} else {
			fmt.Printf("d_closure_2 does not recover\n")
		}
		fmt.Println("d_closure_2 panics")
		panic(fmt.Errorf("d_closure_2 panicked"))

		fmt.Printf("d_closure_2 ends normally\n")
	}()
	defer c()
	defer b()
	//fmt.Println("d panics")
	//panic(fmt.Errorf("d panicked"))

	fmt.Printf("d ends normally\n")
}

func main() {
	fmt.Printf("main starts\n")

	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("main_closure recovered: %v\n", err)
		}
	}()
	d()

	fmt.Printf("main ends normally\n")
}
