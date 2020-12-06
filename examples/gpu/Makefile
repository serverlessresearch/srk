all: vadd vadd.so

vadd.so: vadd.cu
	nvcc vadd.cu -o vadd.so -shared -Xcompiler -fPIC

vadd: vadd.cu
	nvcc vadd.cu -o vadd

clean:
	rm -f vadd
	rm -f vadd.so
