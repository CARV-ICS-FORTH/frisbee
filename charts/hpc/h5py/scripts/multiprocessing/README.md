Demonstrates how to use h5py with the multiprocessing module.
This module implements a simple multi-process program to generate
Mandelbrot set images. It uses a process pool to do the computations,
and a single process to save the results to file.
Importantly, only one process actually reads/writes the HDF5 file.
Remember that when a process is fork()ed, the child inherits the HDF5
state from its parent, which can be dangerous if you already have a file
open. Trying to interact with the same file on disk from multiple
processes results in undefined behavior.
If matplotlib is available, the program will read from the HDF5 file and
display an image of the fractal in a window. To re-run the calculation,
delete the file "mandelbrot.hdf5".