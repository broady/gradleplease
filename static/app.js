var gmsVersion = '4.2.42';
var frameworkVersion = '19.0.1';

var defaultQuery = 'actionbarsherlock';
var previousInputValue = defaultQuery;
var query = defaultQuery;
var sessionId = parseInt(Math.random() * 1e16);

(function() {
  var queryTimeout;
  var s = document.querySelector('#search');
  s.focus();
  s.onchange = s.onkeyup = function() {
    if (previousInputValue == s.value || !s.value) return;
    update('...');
    clearTimeout(queryTimeout);
    queryTimeout = setTimeout(function() {
      previousInputValue = s.value;
      search(previousInputValue);
    }, 200);
  };
  setFeedbackEnabled(true);
  document.querySelector('#didyoumean span').onclick = useSuggestion;
  document.querySelector('#popular').onclick = usePopular;

  window.onkeyup = function(e) {
    if (e.keyCode == 191) { // forward slash
      s.focus();
    }
  };
})();

(function replaceStaticVersions() {
  var gms = document.querySelectorAll('.gmsversion');
  for (var i = 0; i < gms.length; i++) {
    gms[i].innerHTML = gmsVersion;
  }
  var fw = document.querySelectorAll('.frameworkversion');
  for (var i = 0; i < fw.length; i++) {
    fw[i].innerHTML = frameworkVersion;
  }
})();

function usePopular(e) {
  if (e.target.tagName != 'LI') return;
  var s = e.target.textContent;
  document.querySelector('#search').value = s;
  update('...');
  search(s);
  analytics.trackEvent('suggestion', 'popular', s);
}

function useSuggestion() {
  var suggestion = document.querySelector('#didyoumean span').textContent;
  document.querySelector('#search').value = suggestion;
  update('...');
  search(suggestion);
  analytics.trackEvent('suggestion', 'suggestion', suggestion);
}

function feedback(good) {
  return function() {
    setFeedbackEnabled(false);
    var result = document.querySelector('pre span').textContent;
    analytics.trackEvent('feedback', good ? 'good' : 'bad', result, searchesThisSession);
    var img = document.createElement('img');
    img.height = img.width = '1px';
    var q = document.querySelector('#search').value;
    img.src = '/feedback?q=' + encodeURIComponent(q) + '&result=' + encodeURIComponent(result) + '&good=' + good + '&session=' + sessionId;
  };
}

function setFeedbackEnabled(enabled) {
  document.querySelector('#feedback').className = enabled ? '' : 'disabled';
  document.querySelector('#feedback-good').onclick = enabled ? feedback(true) : null;
  document.querySelector('#feedback-bad').onclick = enabled ? feedback(false) : null;
}

var searchesThisSession = 0;

function search(q) {
  if (q.indexOf('play') != -1 || q.indexOf('gms') != -1 || q.indexOf('gcm') != -1) {
    analytics.trackEvent('search', 'overriden', query, ++searchesThisSession);
    update('com.google.android.gms:play-services:' + gmsVersion);
    return;
  }
  if (q.indexOf('compat') != -1) {
    analytics.trackEvent('search', 'overriden', query, ++searchesThisSession);
    update('com.android.support:appcompat-v7:' + frameworkVersion);
    return;
  }
  query = q;
  document.querySelector('#apklibmessage').style.display = 'none';
  setFeedbackEnabled(false);
  analytics.trackEvent('search', 'search', query, ++searchesThisSession);
  if (overrides[q]) {
    query = overrides[q];
    analytics.trackEvent('search', 'overriden', query, searchesThisSession);
  }
  var s = document.createElement('script');
  s.src = '/search?q=' + encodeURIComponent(query) + '&session=' + sessionId;
  s.className = 'search';
  document.body.appendChild(s);
}

function update(s) {
  document.querySelector('#suggestion').style.display = '';
  document.querySelector('#suggestion span').textContent = s;
}

function searchCallback(data) {
  setFeedbackEnabled(true);
  if (data.error) {
    console.log(data.error);
    return sadface();
  }
  if (data.responseHeader && data.responseHeader.params && data.responseHeader.params.q != query) {
    // Response returned out of order
    return;
  }
  if (data.spellcheck && data.spellcheck.suggestions && data.spellcheck.suggestions[1] && data.spellcheck.suggestions[1].suggestion.length) {
    document.querySelector('#didyoumean').style.display = 'block';
    document.querySelector('#didyoumean span').textContent = data.spellcheck.suggestions[1].suggestion[0];
  } else {
    document.querySelector('#didyoumean').style.display = 'none';
  }
  if (!data.response || !data.response.docs || !data.response.docs.length) {
    return sadface();
  }
  var showApklibMessage = false;
  var docs = data.response.docs.filter(function(artifact) {
    var apklibwithaar = artifact.p == 'apklib' && artifact.text.filter(aarFilter).length;
    if (!showApklibMessage && artifact.p == 'apklib' && !apklibwithaar) {
      showApklibMessage = artifact;
    }
    return apklibwithaar || artifact.p != 'pom' && artifact.p != 'apklib' && artifact.a.indexOf('sample') == -1;
  });
  if (showApklibMessage) {
    document.querySelector('#apklibmessage span').textContent = showApklibMessage.id;
    document.querySelector('#apklibmessage').style.display = 'block';
    if (!docs.length) {
      document.querySelector('#suggestion').style.display = 'none';
      return;
    }
  }
  if (!docs.length) {
    return sadface();
  }
  if (docs[0].p == 'apklib') {
    update(docs[0].id + ':' + docs[0].latestVersion + '@aar');
  } else {
    update(docs[0].id + ':' + docs[0].latestVersion);
  }
}

function aarFilter(text) {
  return text.indexOf('aar') != -1;
}

function sadface() {
  update(':(');
}

overrides = {
  'jodatime': 'joda-time',
  'slf4j': 'org.slf4j slf4j-android',
  'slf4j-android': 'org.slf4j slf4j-android',
  'animation': 'com.nineoldandroids library',
  'ormlite': 'com.j256.ormlite ormlite-android',
  'pulltorefresh': 'actionbarpulltorefresh',
  'wire': 'wire-runtime',
  'tape': 'squareup tape',
  'holoeverywhere': 'holoeverywhere library',
  'annotations': 'androidannotations',
  'svg': 'svg-android',
  'commons': 'g:org.apache.commons commons-lang',
  'commons-lang': 'g:org.apache.commons commons-lang',
  'picasso': 'squareup picasso',
  'guava': 'com.google guava''
}
