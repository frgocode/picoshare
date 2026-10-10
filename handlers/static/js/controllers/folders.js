"use strict";

async function handleResponse(response) {
  if (!response.ok) {
    const error = await response.text();
    return Promise.reject(error || response.statusText);
  }
  return response;
}

export async function createFolder(name, parentId) {
  const payload = { name };
  if (parentId) {
    payload.parentId = parentId;
  }
  return fetch("/api/folders", {
    method: "POST",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  })
    .then(handleResponse)
    .then((response) => response.json())
    .catch((error) => {
      if (error.message) {
        return Promise.reject(
          `Failed to communicate with server: ${error.message}.`,
        );
      }
      return Promise.reject(error);
    });
}

export async function renameFolder(id, name) {
  return fetch(`/api/folders/${encodeURIComponent(id)}`, {
    method: "PUT",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ name }),
  })
    .then(handleResponse)
    .catch((error) => {
      if (error.message) {
        return Promise.reject(
          `Failed to communicate with server: ${error.message}.`,
        );
      }
      return Promise.reject(error);
    });
}

export async function moveFolder(id, parentId) {
  const payload = {};
  if (parentId) {
    payload.parentId = parentId;
  } else {
    payload.parentId = null;
  }
  return fetch(`/api/folders/${encodeURIComponent(id)}/move`, {
    method: "PUT",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  })
    .then(handleResponse)
    .catch((error) => {
      if (error.message) {
        return Promise.reject(
          `Failed to communicate with server: ${error.message}.`,
        );
      }
      return Promise.reject(error);
    });
}

export async function deleteFolder(id) {
  return fetch(`/api/folders/${encodeURIComponent(id)}`, {
    method: "DELETE",
    credentials: "include",
  })
    .then(handleResponse)
    .catch((error) => {
      if (error.message) {
        return Promise.reject(
          `Failed to communicate with server: ${error.message}.`,
        );
      }
      return Promise.reject(error);
    });
}

export async function moveFileToFolder(entryId, folderId) {
  const payload = {};
  if (folderId) {
    payload.folderId = folderId;
  } else {
    payload.folderId = null;
  }
  return fetch(`/api/entry/${encodeURIComponent(entryId)}/folder`, {
    method: "PUT",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  })
    .then(handleResponse)
    .catch((error) => {
      if (error.message) {
        return Promise.reject(
          `Failed to communicate with server: ${error.message}.`,
        );
      }
      return Promise.reject(error);
    });
}
