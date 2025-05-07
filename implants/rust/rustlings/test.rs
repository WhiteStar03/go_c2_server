use std::rc::Rc;

// Using Rc::clone

fn main(){
let a = Rc::new(vec![1, 2, 3]);
let b = Rc::clone(&a);  // Just increments reference count
let c = Rc::clone(&a);  // Same data, just another pointer

// All three point to the SAME vector
println!("Reference count: {}", Rc::strong_count(&a));  // 3
println!("{:p}", &*a);  // Same memory address
println!("{:p}", &*b);  // Same memory address
println!("{:p}", &*c);  // Same memory address

// Using normal clone
let x = vec![1, 2, 3];
let y = x.clone();  // Creates a complete copy
let z = x.clone();  // Creates another complete copy

// Each has its own independent vector
println!("{:p}", &x);  // Different memory address
println!("{:p}", &y);  // Different memory address
println!("{:p}", &z);  // Different memory address
}

