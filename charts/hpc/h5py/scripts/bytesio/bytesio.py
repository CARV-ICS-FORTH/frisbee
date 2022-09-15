import io
import h5py

bio = io.BytesIO()
with h5py.File(bio, 'w') as f:
    f['dataset'] = range(10)

data = bio.getvalue() # data is a regular Python bytes object.
print("Total size:", len(data))
print("First bytes:", data[:10])