FROM python:3.11-slim

# Install core libraries for data and plotting
RUN apt-get update && apt-get install -y build-essential libfreetype6-dev libpng-dev && \
    pip install --no-cache-dir fastmcp==2.* pandas numpy scipy matplotlib seaborn plotly && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY python-code-interpreter_server.py /app/server.py
RUN mkdir -p /app/runs

CMD ["python", "server.py"]
