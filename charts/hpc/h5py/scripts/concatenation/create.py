import sys
import os
import h5py
import numpy as np
import socket

def create_random_file(dir, index):
    """create one random file"""
    filename = 'myfile_' + str(index) + "_"+  socket.gethostname() + ".h5"
    name = os.path.join(dir, filename)

    f = h5py.File(name=name, mode='w')

    d = f.create_dataset('data', (5, 10, 20), 'i4')
    data = np.random.randint(low=0, high=100, size=(5*10*20))
    data = data.reshape(5, 10, 20)
    d[:] = data

    print("Flushing dataset") # That used to be missing, resulting into erroneous behaviors.
    d.flush()

    print("Closing HDF5 file") # That used to be missing, resulting into erroneous behaviors.
    f.flush()
    f.close()

    return name


def main(argv):
    if len(argv) != 3:
        print("Inputs: [shared_dir] [num_of_files]")
        return

    shared_dir = argv[1]
    num_of_files = int(argv[2])

    print("ShareDir:", shared_dir, " num_of_files:", num_of_files)

    for i_file in range(num_of_files):
            name = create_random_file(dir=shared_dir, index=i_file)
            print("Creating ", name)


if __name__ == '__main__':
    main(sys.argv)