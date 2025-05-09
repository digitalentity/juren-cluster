# Concepts

## Network

* A Cluster is built of Nodes. All Nodes belong to the network where unrestricted multicast communication is possible.

## Data

All material data in JUREN is stored as a Block or a sequence of Blocks.

### OID

An Object Identifier, or OID, is a label used to point to data in JUREN. It doesn't indicate where the content is stored, but it forms a kind of address based on the content itself (in most cases). OIDs are short, regardless of the size of their underlying content.

In some cases OID is not based on the entirety of the object content, but rather the part of it.

OID includes the type of the content:

    - Raw Block (pure data)
    - Object
    - Object Set

OID structure:

    - Version (1 byte)
    - Padding/Reserved (1 byte)
    - Type (1 byte)
    - SHA256 hash (32 bytes)

The OID size is 35 bytes.

### Blocks

A Block is a unit of storage. Each Block is identified by its OID.

Block has the following Metadata

    * OID
    * Size
    * Creation Time
        - Nanosecond-accurate time since Unix Epoch, records the time when the Block was created.
    * Tombstone Time
        - Nanosecond-accurate time since Unix Epoch, records the time when the Block was deleted.
    * Replication Factor
        - Min and Max number of Nodes that should have the block stored and made available

### Objects

An Object (or BlockSet) is an ordered collection of blocks. An Object itself is stored as a Block.

An Object stores the following Data:

    * Ordered list of blocks
        - In memory this is organized as a tree, indexed by offset, to allow fast seeking.

### Object Sets

An Object Set (ObsetSet) is a collection of named objects.

An ObjectSet stores the following dat:
    * ObjectSet name (namespace)
    * List of Items. Each Item has:
        - a path, which could be used to represent the Items place in a virtual filesystem
        - a OID of an Object that stores the actual data.

## Block Index

Each node has its own Key/Value storage of Block Metadata keyed by OID.

Each Note periodically announces the entire content of the Block Index.

## Replication

Each Node 


## Garbage collection


# Scalability

The size of the block metadata is approximately 100 bytes (for simplicity of calculation).

With the block size of 256K, the number of index blocks required to store the amount of data is as follows:

| Data Size | Number of Blocks | Index Size | Time to transfer the index at 1 Gbit/s
---|---|---|---
1 TB | 4,000,000 | 400 MB | 3.2 seconds
10 TB | 40,000,000 | 4 GB | 32 seconds
100 TB | 400,000,000 | 40 GB | 5.3 minutes
1 PB | 4,000,000,000 | 400 GB | 53 minutes


With the block size of 1M, the number of index blocks required to store the amount of data is as follows:

| Data Size | Number of Blocks | Index Size | Time to transfer the index at 1 Gbit/s
---|---|---|---
1 TB | 1,000,000 | 100 MB | 800 ms
10 TB | 10,000,000 | 1 GB | 8 seconds
100 TB | 100,000,000 | 10 GB | 80 seconds
1 PB | 1,000,000,000 | 100 GB | 13.3 minutes

