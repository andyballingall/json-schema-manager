# JSON Schema Manager: Registry <!-- omit in toc -->

- [Introduction](#introduction)
- [Registry Concepts](#registry-concepts)
  - [Schema Registry Configuration File](#schema-registry-configuration-file)
  - [Namespacing](#namespacing)
  - [Immutability and Schema ID](#immutability-and-schema-id)
  - [Schema Family and Semantic Versioning](#schema-family-and-semantic-versioning)
  - [Directory Structure](#directory-structure)
  - [Filename Follows Directory Structure](#filename-follows-directory-structure)
  - [Identity Follows Directory Structure](#identity-follows-directory-structure)
  - [Special Properties](#special-properties)
    - [$id](#id)
    - [$ref](#ref)
  - [Visibility Control](#visibility-control)
- [Creating a new Schema](#creating-a-new-schema)
- [CI/CD Workflows](#cicd-workflows)


## Introduction
A JSON Schema Registry is a directory under the management of the JSON Schema Manager (`jsm`), containing every version of every JSON schema your organisation uses, along with suites of test documents which are used by JSON Schema Manager to validate that the schemas work as intended. Place your schema registry in a git repository, and use the `jsm` command to manage the registry, run tests, and render production schemas for both private and public consumption.

JSON Schema Manager requires the root of the registry to contain a [Schema Registry Configuration File](#schema-registry-configuration-file).

## Registry Concepts

### Schema Registry Configuration File

A directory is a schema registry if it contains a file named `json-schema-manager-config.yml`.

E.g. 
```yaml
environments:
  dev:
    privateUrlRoot: "https://dev.json-schemas.internal.myorg.io/"
    publicUrlRoot: "https://dev.json-schemas.myorg.io/"
    allowSchemaMutation: true
  uat:
    privateUrlRoot: "https://uat.json-schemas.internal.myorg.io/"
    publicUrlRoot: "https://uat.json-schemas.myorg.io/"
  prod:
    privateUrlRoot: "https://json-schemas.internal.myorg.io/"
    publicUrlRoot: "https://json-schemas.myorg.io/"
```

This file defines the following:
- the environments that schemas can be published to
- the private URL root for each environment
- the public URL root for each environment
- whether schemas can be mutated in each environment. By default, schemas cannot be changed once published, but for specific development environments, this can be overriden with the `allowSchemaMutation` property.

By default, schemas are private. See [Visibility Control](#visibility-control) for more information.

### Namespacing

In order to avoid naming conflicts between different parts of an organisation, schemas are namespaced. This is achieved by creating one or more namespace directories at the root of the registry, for example:

```
logistics/data/            # Logistics domain data pipeline schemas
customer/loyalty/          # Customer loyalty schemas
common/                    # Common schemas shared across the organisation
```
> Directory names must be lowercase, and only contain the following characters: `a-z`, `0-9`, and `-`.


### Immutability and Schema ID

All production JSON Schemas managed by JSON Schema Manager are intended to be immutable. This means that once a schema is published to a production environment, it **cannot** be changed. This gives confidence to users of the schemas that a given semantic version of a schema family will never change.

Idiomatically, the ID of a JSON Schema (defined by the `$id` property of the schema) is a URL which can be used to:

- Uniquely identify the schema globally.
- Download the schema itself.

Note that it **is** possible, and beneficial, to republish changes to a schema in a **non-production** environment. This allows for teams to tweak a proposed schema numerous times during development without having to create a new version of the schema each time. See the `allowSchemaMutation` property in [Schema Registry Configuration File](#schema-registry-configuration-file).

### Schema Family and Semantic Versioning

A schema family is a collection of versions of a schema for a specific purpose. For example, a schema family may represent a customer record sent over a message queue. 

It is common that the data validated by a JSON Schema changes over time. For example, the customer record may have a new field added to support a new feature.

JSON Schema Manager addresses this by forcing the creation of a **family** of versions of the schema. Each new addition to the family is immutable (i.e. once published to production, it cannot be changed). Schema families use [semantic versioning](https://semver.org/) as follows:

- use a MAJOR version change to indicate a breaking change
- use a MINOR version change when adding new features in a backward-compatible manner
- use a PATCH version change when making non-breaking non-feature changes, such as relaxing a value constraint.

For example, consider a customer schema:

1. Version `1.0.0` of a customer schema may have properties `id`, `firstName`, `lastName`. 
2. Version `1.1.0` may add a `dateOfBirth` property.
3. Version 2.0.0 may change the structure to replace `firstName` with an array of `givenNames`, and `lastName` with `familyName` - i.e. a breaking change
4. Version `2.0.1` may augment the description of the properties in the schema to provide additional context.

### Directory Structure

Each JSON Schema is stored in separate dedicated **schema home directory**. The location of a schema's home directory is: 

```
COMPONENT                             EXAMPLE
-------------------------------------------------------------------------------
<Registry Root Directory>/            /path/to/registry/
 <Namespace Directory(s)>/            logistics/data/
  <Schema Family Directory>/          shipment/
   <3 semantic version directories>/  1/2/3/
``` 

Where:

1. `<Registry Root Directory>`: is a directory containing every schema managed by JSON Schema Manager - e.g. `/path/to/schemas`
2. `<Namespace Directory(s)>`: is one or more namespacing directories - e.g. `logistics/data`
3. `<Schema Family Directory>`: is a directory describing the purpose of a specific schema family - e.g. `shipment`. All versions of the schema (separate schemas) are stored under this directory.
4. `<Semantic Version>`: is 3 directories describing a semantic version of a specific version of as schema family (`MAJOR/MINOR/PATCH` - e.g. `1/2/3`). The patch folder acts effectively as the schema's home directory, containing:

Within the schema's home directory you will find:
1. The schema itself (ending in `.schema.json`)
2. `pass` - a folder containing JSON documents which we expect the schema to validate
3. `fail` - a folder containing JSON documents which we expect the schema to **fail** to validate.
4. An optional `README.md` - a markdown file containing extra context for the schema which is not part of the schema itself.


So the complete path for a schema with filename `logistics_data_shipment_1_2_3.schema.json` in the registry might be: 

```
/path/to/registry/logistics/data/shipment/1/2/3/logistics_data_shipment_1_2_3.schema.json
```

### Filename Follows Directory Structure

The filename of JSON Schemas within the registry **must** match the directory structure of the schema's path within the registry. JSON Schema Manager will enforce this rule. This ensures that a schema is easy to find, and that published schema URLs are guaranteed not to collide.

E.g. this is the path of a schema within a JSON Schema Manager Registry:

```
customer/b2c-customer/1/0/0/customer_b2c-customer_1_0_0.schema.json
```

The `_` character is used to separate components of the directory path in the filename of the schema.


### Identity Follows Directory Structure

The `$id` property of a JSON Schema **must** match the directory structure. JSON Schema Manager will auto-generate the ID based on the directory structure, the environment being published to, and the settings in the registry configuration file in the root of the registry: `json-schema-manager-config.yml`

See [Special Properties](#special-properties) below for more details.

### Special Properties

#### $id

Always set as follows:

```json
"$id": "{{ ID }}"
```
This will be set automatically by the `jsm` command to the correct URL.

The `$id` property is used to identify the schema, and is the URL that will be used to reference the schema in other schemas. For published schemas, it is the URL that will be used to **load** the schema.

JSON Schema Manager will automatically set the `$id` property to the correct URL based on:
- the namespace, the schema family, the semantic version of the schema, 
- the environment being published to, 
- whether the schema is public or private (see [Visibility Control](#visibility-control) below), 
- the settings in the [Registry Configuration File](#registry-configuration-file).

#### $ref

If you are using `$ref` to reference another schema managed by JSON Schema Manager, use the following format with the **filename** of the schema you want to reference. This will automatically be converted into the correct URL on publication.

```json
"$ref": "{{ JSM `<schema filename>` }}"
```
e.g.

```json
"$ref": "{{ JSM `customer_b2c-customer_2-0-0.schema.json` }}"
```

If referencing schema within the same file, or an external schema not managed by JSON Schema Manager, just set `$ref` in the usual way defined in the JSON Schema specification.

### Visibility Control

By default, all schemas are considered **private** and are only published to an internal-only location. 

To make a schema available at a **public** URL, add the `x-public` property to the root of the JSON schema:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "{{ ID }}",
  "title": "My Public Schema",
  "x-public": true,
  "type": "object",
  ...
}
```

The `x-public` property is a boolean that indicates whether the schema should be published to a public URL. If `x-public` is missing or set to `false`, the schema will remain private.

Configuration of the location of both private and public schemas is done in the configuration file json-schema-manager-config.yml at the root of the schema registry passed to the 'jsm' command.

## Creating a new Schema

Use the `jsm` command to create a new schema.

```bash
jsm new-schema <destination directory path> [--dialect <dialect>] [--create-dirs]
```
e.g.

```bash

jsm new-schema schemas/demo/domain-a/entity-a/1/2/3

jsm new-schema schemas/demo/domain-a/domain-a-b/entity-a/1/2/3 --dialect draft-2020-12 --create-dirs
```

The command will create a new schema file at the specified destination path. If you use the `--create-dirs` flag, it will create any missing parent directories.

The `--dialect` flag will specify the dialect of the schema to create. If not specified, it will default to `draft-07`, which is widely supported and reasonably expressive.

Valid dialects are:
- `draft-04`
- `draft-06`
- `draft-07` (default)
- `draft-2019-09`
- `draft-2020-12`

## CI/CD Workflows

Use the `jsm` command to publish a schema in your schema registry repo as part of your CI/CD pipeline.

Use `jsm publish --env <env name> --file <schema filename>`

Note that if publishing to an environment which is a production environment, `jsm` will test for the existence of an extant schema at the URL defined by the `$id` property of the schema. If the schema already exists, `jsm` will fail the publish and not allow the schema to be published.

