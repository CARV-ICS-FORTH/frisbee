FROM python:3.7-slim as final

# The build stage installs everything into the virtualenv, and the final stage just copies the virtualenv over into a small image.
RUN python -m venv /venv

RUN pip install --upgrade pip flwr setuptools wheel


RUN pip install --upgrade torch===1.10.2
RUN pip install --upgrade torchvision===0.11.3

# Install dependencies under the user home directory
RUN apt-get update && apt-get install -y git

# Clone the python code
RUN git clone --depth=1 https://github.com/adap/flower.git && \
    mv flower/examples/quickstart_pytorch .

# Do stuff on the code
WORKDIR quickstart_pytorch


# Copy a modified version of the client
#ENV FL_SERVER "127.0.0.1"
#ENV FL_DATASET "./dataset"

COPY client.py ./client.py