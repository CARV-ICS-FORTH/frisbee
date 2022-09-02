import sys
from os import walk
import h5py
import numpy as np


def concatenate(shared_dir, file_names_to_concatenate):
    entry_key = 'data'  # where the data is inside the source files.

    ref_file = shared_dir + "/" + file_names_to_concatenate[0]
    print("Use as reference file ", ref_file)

    sh = h5py.File(ref_file, 'r')[entry_key].shape  # get the first ones shape.

    layout = h5py.VirtualLayout(shape=(len(file_names_to_concatenate),) + sh,
                                dtype=np.float64)

    output_file = shared_dir + "/" + "VDS.h5"

    f = h5py.File(output_file, 'w', libver='latest')
    print("Writing to ", output_file)

    # Append files to the reference file
    for i, filename in enumerate(file_names_to_concatenate):
        vsource = h5py.VirtualSource(filename, entry_key, shape=sh)
        layout[i, :, :, :] = vsource

    f.create_virtual_dataset(entry_key, layout, fillvalue=0)

    print("Closing HDF5 file")  # That used to be missing, resulting into erroneous behaviors.
    f.flush()
    f.close()


def main(argv):
    if len(argv) != 2:
        print("Inputs: [shared_dir]")
        return

    shared_dir = argv[1]

    # return all files in the directory
    files = next(walk(shared_dir), (None, None, []))[2]  # [] if no file

    # filter the list to include only .h5 files
    files = [x for x in files if x.endswith('.h5')]

    print("Concatenating files from:", shared_dir, " list:", files)

    # merge files
    concatenate(shared_dir, files)


if __name__ == '__main__':
    main(sys.argv)
