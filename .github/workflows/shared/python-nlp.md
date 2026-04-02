---
# Python NLP Environment - scikit-learn, NLTK, TextBlob, WordCloud
# Provides TF-IDF vectorization, K-means clustering, sentiment analysis, and NLP utilities

tools:
  bash:
    - "*"

network:
  allowed:
    - python

steps:
  - name: Setup Python NLP environment
    run: |
      mkdir -p /tmp/gh-aw/python/{data,charts,artifacts}
      # Create a virtual environment for proper package isolation (avoids --break-system-packages)
      if [ ! -d /tmp/gh-aw/venv ]; then
        python3 -m venv /tmp/gh-aw/venv
      fi
      echo "/tmp/gh-aw/venv/bin" >> "$GITHUB_PATH"
      /tmp/gh-aw/venv/bin/pip install --quiet nltk scikit-learn textblob wordcloud

      # Download required NLTK corpora
      /tmp/gh-aw/venv/bin/python3 -c "
      import nltk
      for corpus in ['punkt_tab', 'stopwords', 'vader_lexicon', 'averaged_perceptron_tagger_eng']:
          nltk.download(corpus, quiet=True)
      print('NLTK corpora ready')
      "

      /tmp/gh-aw/venv/bin/python3 -c "import sklearn; print(f'scikit-learn {sklearn.__version__}')"
---

## Python NLP Environment Ready

Libraries: scikit-learn, NLTK, TextBlob, WordCloud
Directories: `/tmp/gh-aw/python/{data,charts,artifacts}`

### TF-IDF + K-means Clustering Pattern

```python
from sklearn.feature_extraction.text import TfidfVectorizer
from sklearn.cluster import KMeans
import numpy as np

# Vectorize text
vectorizer = TfidfVectorizer(max_features=500, stop_words='english', ngram_range=(1, 2))
X = vectorizer.fit_transform(texts)

# Find optimal k using elbow method
inertias = []
k_range = range(2, min(11, len(texts)))
for k in k_range:
    km = KMeans(n_clusters=k, random_state=42, n_init=10)
    km.fit(X)
    inertias.append(km.inertia_)

# Fit final model
optimal_k = 5  # or determine from elbow
kmeans = KMeans(n_clusters=optimal_k, random_state=42, n_init=10)
labels = kmeans.fit_predict(X)

# Label clusters by top TF-IDF terms
feature_names = vectorizer.get_feature_names_out()
for cluster_id in range(optimal_k):
    center = kmeans.cluster_centers_[cluster_id]
    top_terms = [feature_names[i] for i in center.argsort()[-10:][::-1]]
    print(f"Cluster {cluster_id}: {', '.join(top_terms)}")
```

### Sentiment Analysis Pattern (TextBlob / VADER)

```python
from textblob import TextBlob
from nltk.sentiment.vader import SentimentIntensityAnalyzer

# TextBlob polarity (-1 negative, +1 positive)
blob = TextBlob(text)
polarity = blob.sentiment.polarity

# VADER compound score (-1 to +1)
sia = SentimentIntensityAnalyzer()
scores = sia.polarity_scores(text)
compound = scores['compound']  # >= 0.05 positive, <= -0.05 negative
```

### Text Preprocessing

```python
import re
from nltk.corpus import stopwords

stop_words = set(stopwords.words('english'))

def clean_text(text):
    text = re.sub(r'```.*?```', '', text, flags=re.DOTALL)  # Remove code blocks
    text = re.sub(r'http\S+', '', text)                      # Remove URLs
    text = re.sub(r'[^a-zA-Z\s]', ' ', text)                # Keep only letters
    tokens = text.lower().split()
    return ' '.join(t for t in tokens if t not in stop_words and len(t) > 2)
```

### Best Practices

- Use JSON Lines (`.jsonl`) for append-only historical storage
- Cache vectorizers to avoid re-fitting on the same data
- Label clusters by top TF-IDF terms
- Use VADER for short social text; TextBlob for longer prose
