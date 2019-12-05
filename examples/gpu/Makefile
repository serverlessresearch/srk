all: vadd vadd.so

vadd.so: vadd.cu
	nvcc vadd.cu -o vadd.so -shared -Xcompiler -fPIC -lcublas

vadd: vadd.cu
	nvcc vadd.cu -o vadd
