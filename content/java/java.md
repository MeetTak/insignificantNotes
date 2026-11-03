---
title: "Java"
folder: "Java"
created: "2024-11-05T18:45:42.000Z"
modified: "2026-04-21T05:21:19.000Z"
note_id: "x-coredata://8788A586-C8FC-4484-AB68-760AF85E0359/ICNote/p2081"
source: "apple-notes"
---

Math.sqrt()

System.exit(0)

Class name and interface name cannot be same

Thread class in the java.lang package extends the runnable interface

The start() method only invokes the run method present in the class if we have any other method present in the class we need to explicitly call it

new MyRunnable() creates the new instance of the MyRunnable() class

printStackTrace() is a method in Throwable class in java, which is the superclass of all the errors and exceptions in the java language.

new()

For the next two points
Bird bird = new Bird;
Bird: This is the class name.
bird: This is the variable name that will hold the reference to the new Bird object.
new Bird(): This is the constructor call that creates a new instance of the Bird class.

Animal animal = bird
Since Bird extends abstract class Animal, so Bird instance can be used to define a variable of type Animal

Trackable tracked = bird
Since Bird implements interface Trackable, so Bird instance can be used to define a variable of type Animal

Constants are usually declared with the help of static final keywords, but in an interface, fields are implicitly public, final and static; it means that we can use the constant declared inside the interface without creating the instance of the interface.
Example
interface FlightEnabled
We declared the constant
double KM_TO_MILES = 0.62;

And in the Main class which is not implementing the interface u can use that
double miles = kilometers \* FlightEnabled.KM_TO_MILES;

This checks if the flier object is an instance of the Trackable interface, if this is true then the flier object is cast to the Trackable type and assigned to the variable tracked. This allows u to safely use variable tracked inside of the if block; otherwise u will get error.

Abstract classes can not be instantiated directly but we can use it indirectly to instantiate instance of that abstract class.

Although its necessary to implement all the abstract methods but still if u don’t declare all the abstract methods in ur class then u must declare ur class as abstract as well.

flier instanceof Tracked tracked
Instanceof is used to check whether the object flier is an instance of a specific class or implements a specific interface.

## In Java, \`varArgs\` (variable-length arguments) allow you to pass an arbitrary number of arguments to a method. This is useful when you don't know in advance how many arguments will be passed. The syntax for \`varArgs\` is to use three dots (\`...\`) after the type in the method parameter.

Here is an example from your code:

\`\`\`java
private static void printText(String... textList) {
for (String s : textList) {
System.out.println(s);
}
}
\`\`\`

\### Explanation:
\- \`String... textList\`: This means the method \`printText\` can accept zero or more \`String\` arguments.
\- Inside the method, \`textList\` is treated as an array of \`String\` objects.

You can call this method with any number of \`String\` arguments, including none:

\`\`\`java
printText("Hello", "World", "again");
printText("Hello");
printText();
\`\`\`

In each case, \`textList\` will be an array containing the passed arguments.

The System.out.println works differently for arrays and String objects as for arrays it prints the memory address of the first element whereas for Strings its directly gonna print the String which u have stored as a element inside the array.

Stream pipelines will work without intermediate operations but it won’t work without terminal operations will give an error.

->Primitive Streams
IntStreams, LongStreams and DoubleStreams

Benefits
Avoid boxing and unboxing.
Provide specialized methods for numeric operations(sum, avg, etc).

Unbounded wildcard (?):
Accepts any type.
Example: List\<?> can hold a list of any type.

Upper-bounded wildcard (? extends Type):
Accepts Type or any of its subclasses.
Example: List\<? extends Number> can hold List\<Integer>, List\<Double>, etc.

Lower-bounded wildcard (? super Type):
Accepts Type or any of its superclasses.
Example: List\<? super Integer> can hold List\<Integer>, List\<Number>, or List\<Object>.

Lambda expression works when we have a functional interface
@FunctionalInterface Attribute(interface with exactly one abstract method
This is incorrect
Thread t1 = () -> matrix.display();
Bcoz Thread is a concrete class not a functional interface
This is correct
Thread t1 = new Thread(() -> matrix.display());
Alternatively,
Runnable r = () -> matrix.display();
Thread t1 = new Thread(r);

There are two ways to start a new thread

Method reference, it is a short hand for lambda expressions which can be use to call the methods by referencing it the class directly.
[https://www.geeksforgeeks.org/method-references-in-java-with-examples/](https://www.geeksforgeeks.org/method-references-in-java-with-examples/)

In java, \\s+ represents one or more whitespace characters, whereas \\s represents a single whitespace character. In java string backslash is an escape character, so to represent a string literal backslash in a string, u need to use \\\\. Therefore, \\s+ becomes \\\\s+ and \\s becomes \\\\s.

Semaphores, locks, and countDownLatches
// Using a CountDownLatch to ensure t2 starts first
CountDownLatch latch = new CountDownLatch(1);

class OrderedTask implements Runnable {
private CountDownLatch latch;
private boolean isT2;
public OrderedTask(CountDownLatch latch, boolean isT2) {
this.latch = latch;
this.isT2 = isT2;
}
public void run() {
if (!isT2) {
try {
latch.await(); // t1 waits for t2 to start
} catch (InterruptedException e) {
e.printStackTrace();
}
} else {
latch.countDown(); // t2 signals it's starting
}
// rest of the task logic
}
}

What is StringBuffer?
StringBuffer is a class that is used to create mutable sequence of characters. Unlike the String class that is immutable.

In the while loop the input.readLine() method is a blocking method as it does not allow server to accept the request from other clients.

### I/O

Overview

- [Byte Streams](https://docs.oracle.com/javase/tutorial/essential/io/bytestreams.html) handle I/O of raw binary data.
- [Character Streams](https://docs.oracle.com/javase/tutorial/essential/io/charstreams.html) handle I/O of character data, automatically handling translation to and from the local character set.
- [Buffered Streams](https://docs.oracle.com/javase/tutorial/essential/io/buffers.html) optimize input and output by reducing the number of calls to the native API.
- [Scanning and Formatting](https://docs.oracle.com/javase/tutorial/essential/io/scanfor.html) allows a program to read and write formatted text.
- [I/O from the Command Line](https://docs.oracle.com/javase/tutorial/essential/io/cl.html) describes the Standard Streams and the Console object.
- [Data Streams](https://docs.oracle.com/javase/tutorial/essential/io/datastreams.html) handle binary I/O of primitive data type and String values.
- [Object Streams](https://docs.oracle.com/javase/tutorial/essential/io/objectstreams.html) handle binary I/O of objects.

Character streams are often "wrappers" for byte streams. The character stream uses the byte stream to perform the physical I/O, while the character stream handles translation between characters and bytes.

Data Streams
note that data streams check for end of file condition by caching EOFException, instead of testing for invalid value.

DataStreams uses one very bad programming technique: it uses floating point numbers to represent monetary values. In general, floating point is bad for precise values. It's particularly bad for decimal fractions, because common values (such as 0.1) do not have a binary representation

The correct type to use for currency values is [java.math.BigDecimal](https://docs.oracle.com/javase/8/docs/api/java/math/BigDecimal.html). Unfortunately, BigDecimal is an object type, so it won't work with data streams. However, BigDecimal *will* work with object streams

Object Streams

If readObject() doesn't return the object type expected, attempting to cast it to the correct type may throw a [ClassNotFoundException](https://docs.oracle.com/javase/8/docs/api/java/lang/ClassNotFoundException.html)

**Note**: You might wonder what happens if two objects on the same stream both contain references to a single object. Will they both refer to a single object when they're read back? The answer is "yes." A stream can only contain one copy of an object, though it can contain any number of references to it. Thus if you explicitly write an object to a stream twice, you're really writing only the reference twice. For example, if the following code writes an object ob twice to a stream

eg
Object ob = new Object();
out.writeObject(ob);
out.writeObject(ob);

will look like that u created two diff obj point to the same reference obj
Object ob1 = in.readObject();
Object ob2 = in.readObject();

we use the below command when we want to terminate the execution of the program at any instance
System.exit(0)

[https://graalvm.github.io/native-build-tools/latest/gradle-plugin.html#adding-the-plugin](https://graalvm.github.io/native-build-tools/latest/gradle-plugin.html#adding-the-plugin)

Binary Tree
internal nodes are those nodes that has atleast one child node.

toString
only looks at first level-print array references as object hashes

deepToString
recursively digs into nested arrays and prints actual contents

instanceof is a java parameter that checks if an object is an instance of a specific class or interface
if(ele instanceof Object\[\]) {
System.out.println(“element toString: “ Arrays.deepToString((Object\[\]) ele));
}
ele is declared as an object
Arrays.deepToString requires an Object\[\] and then cast the ele into Object\[\], otherwise it will give an error ele is not an object i.e. belongs to a specific class or interface.

Random random = new Random();
here the first random means class type
random is the class type
new Random() creates the new instance of the random class
the above is an example for object variable

example for primitive variable
int age = 10;

String\[\] args
array parameter, accepts an array of string objects

String… args
varargs(variable length argument) parameter, accepts zero or more string arguments

jvm performance enhance using dockerfile
openj9
graalvm
fabric8 shell

logging in spring boot plays a vital role in spring boot applications for recording information, actions, and events within the app. it is used for monitoring the performance of an application, understanding the behavior of the application.

The **@Bean Annotation in Spring is used to define a method that produces a bean to be managed by the Spring IOC (Inversion of Control) container.** When we annotate a method with @Bean, we are telling Spring to manage this object (The method will return an instance of a class), and Spring will treat that instance as a bean in its container. These beans can be injected into other components such as **service, controllers, or repositories** through dependency injection.

types of static members in java
static variable(or class variable)
static block
static method
static nested classes

we can pass arguments to the program in this manner as well  echo "1\\n2\\n3" | java OOPS.practice.Main

executor framework provides high lvl api for managing the threads

to manage the java project env setting using sdk, use the following commands
sdk env init
cat .sdkmanrc
to list the env settings of the project

.m2 is maven directory which stays inside the home directory on ur machine

generics basic properties
type erasure: java removes generic type info at runtime. generics only exist at compile-time. the actual code works with object.
memory efficiency: primitives are stored directly in memory, while obj are references. generics were designed to work with objs, not raw values.
consistency: collections like List\<T>, Map\<K, V>, etc., all follow this rule. u can’t do List\<int>, only List\<Integer>.
bottom line: its a core limitation of java generic architecture. if u need to work with primitive, use their wrapper classes or write separate non-generic overloaded methods for each primitive type.

object oriented programming is info hiding

java has a strong type system, very fast compiler, modularity, rigidity, good info hiding, and privacy controls
