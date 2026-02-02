- [JSON Schema Manager (jsm)](#json-schema-manager-jsm)
  - [Why JSON Schema?](#why-json-schema)
  - [JSON Schema Manager Registry](#json-schema-manager-registry)
  - [Developing Schemas](#developing-schemas)
  - [Contributing](#contributing)


# JSON Schema Manager (jsm)

![Build Status](https://github.com/andyballingall/json-schema-manager/actions/workflows/ci.yml/badge.svg)
![Coverage Status](https://github.com/andyballingall/json-schema-manager/wiki/coverage.svg)

JSON Schema Manager (JSM) is a framework that provides a robust workflow and firm foundations for organisations which need to develop, test, deploy, and evolve JSON Schemas over time.

JSM provides a CLI tool (`jsm`) that operates on a repo containing all the versions of every JSON schema in an organisation. 

It:

- Lays down a firm foundation of rules about what constitutes a valid organisation schema. 
- Provides an incredibly simple schema testing environment which allows people with little programming knowledge to define and evolve JSON schemas
- Validates that a supposed 'non-breaking' change to a schema is in fact non-breaking
- is production-environment aware, and can generate rendered versions of schemas which can be deployed into a production or non-prod environment.

## Why JSON Schema?

The data that flows through an organision naturally takes the form of hierarchical structured data - often JSON documents, or JSON-like documents (YAML, TOML, etc). Converting between this structured data and data structures within code is straightforward. //Portable// validation of data at rest, or in transit, is a key capability. It allows for the decoupled development, testing, and deployment of individual components within a distributed architecture. It supports the early identification of bad data moving through a system, reducing the change of bad data disrupting a business. It allows document-centric storage to be used in place of relational databases, removing the need to build costly, brittle, and slow, implementations. 

Unlike relational SQL models, JSON Schemas provide an effective way to define portable data contracts for data at rest, moving through an API or event-driven architecture. And unlike relational SQL models, which only have validation power from within the context of a database server, JSON Schemas are well-supported by a wide range of tools and languages, and can be used to validate data within any component. 

## JSON Schema Manager Registry

JSON Schema Manager operates on a registry of JSON Schemas, which we recommend are stored in a git repository. This registry is highly structured. For further details, see [JSON Schema Manager Registry](./user-docs/registry.md).

## Developing Schemas

For details on how to develop and testschemas with JSM, see [Developing Schemas](./user-docs/developing-schemas.md).

## Contributing

If you would like to contribute to the development of JSM, please see [CONTRIBUTING.md](./CONTRIBUTING.md) for environment setup and build instructions.
