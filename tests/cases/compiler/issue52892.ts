// @strict: true

class A<T = number> {
    //static { type _<T extends A = A> = 0 }

    value!: T;
    child!: InstanceType<typeof A.B<A<T>>>

    static B = class B<T extends A = A> {
        parent!: T;
    } 
}

var a = new A
a.child.parent.value
