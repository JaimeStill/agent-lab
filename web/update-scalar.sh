#!/bin/bash
set -e

cd "$(dirname "$0")"

npm i @scalar/api-reference

cp node_modules/@scalar/api-reference/dist/browser/standalone.js docs/scalar.js
cp node_modules/@scalar/api-reference/dist/style.css docs/scalar.css

rm -rf node_modules package-lock.json

echo "Scalar assets updated successfully"
