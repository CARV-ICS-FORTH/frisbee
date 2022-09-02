import sys
import h5py


def visualize_file():
    """ Open the HDF5 file and display the result """
    try:
        import matplotlib.pyplot as plt
    except ImportError:
        print("Whoops! Matplotlib is required to view the fractal.")
        raise

    f = h5py.File('/testdata/mandelbrot.hdf5', 'r')
    dset = f['mandelbrot']
    a = dset[...]
    plt.imshow(a.transpose())

    print("Saving fractal at /testdata/mygraph.png.")
    plt.savefig("/testdata/mygraph.png")

    print("Displaying fractal. Close window to exit program.")
    try:
        plt.show()
    finally:
        f.close()


if __name__ == '__main__':
    if not h5py.is_hdf5('/testdata/mandelbrot.hdf5'):
        print('Not an HDF5 file')
        sys.exit(-1)
    else:
        visualize_file()
