import multiprocessing as mp

import h5py
import numpy as np

# === Parameters for Mandelbrot calculation ===================================

NX = 512
NY = 512
ESCAPE = 1000

XSTART = -0.16070135 - 5e-8
YSTART = 1.0375665 - 5e-8
XEXTENT = 1.0E-7
YEXTENT = 1.0E-7

xincr = XEXTENT * 1.0 / NX
yincr = YEXTENT * 1.0 / NY


# === Functions to compute set ================================================

def compute_escape(pos):
    """ Compute the number of steps required to escape from a point on the
    complex plane """
    z = 0 + 0j;
    for i in range(ESCAPE):
        z = z ** 2 + pos
        if abs(z) > 2:
            break
    return i


def compute_row(xpos):
    """ Compute a 1-D array containing escape step counts for each y-position.
    """
    a = np.ndarray((NY,), dtype='i')
    for y in range(NY):
        pos = complex(XSTART, YSTART) + complex(xpos, y * yincr)
        a[y] = compute_escape(pos)
    return a


# === Functions to run process pool & visualize ===============================

def run_calculation():
    """ Begin multi-process calculation, and save to file """

    print("Creating %d-process pool" % mp.cpu_count())

    pool = mp.Pool(mp.cpu_count())

    f = h5py.File('/testdata/mandelbrot.hdf5', 'w')

    print("Creating output dataset with shape %s x %s" % (NX, NY))

    dset = f.create_dataset('mandelbrot', (NX, NY), 'i')
    dset.attrs['XSTART'] = XSTART
    dset.attrs['YSTART'] = YSTART
    dset.attrs['XEXTENT'] = XEXTENT
    dset.attrs['YEXTENT'] = YEXTENT

    result = pool.imap(compute_row, (x * xincr for x in range(NX)))

    for idx, arr in enumerate(result):
        if idx % 25 == 0: print("Recording row %s" % idx)
        dset[idx] = arr

    print("Closing HDF5 file")

    f.close()

    print("Shutting down process pool")

    pool.close()
    pool.join()


if __name__ == '__main__':
    if not h5py.is_hdf5('/testdata/mandelbrot.hdf5'):
        run_calculation()
    else:
        print('Fractal found in "mandelbrot.hdf5". Delete file to re-run calculation.')
