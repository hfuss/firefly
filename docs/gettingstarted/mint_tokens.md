---
layout: default
title: Mint some tokens
parent: Getting Started
nav_order: 9
---

# Mint some tokens
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

## Quick reference

Tokens are a critical building block in many blockchain-backed applications. Fungible tokens can represent a store
of value or a means of rewarding participation in a multi-party system, while non-fungible tokens provide a clear
way to identify and track unique entities across the network. FireFly provides flexible mechanisms to operate on
any type of token and to tie those operations to on- and off-chain data.

- FireFly provides an abstraction layer for multiple types of tokens
- Tokens are grouped into _pools_, which each represent a particular type or class of token
- Each pool is classified as _fungible_ or _non-fungible_
- In the case of _non-fungible_ tokens, the pool is subdivided into individual tokens with a unique _token index_
- Within a pool, you may _mint (issue)_, _transfer_, and _burn (redeem)_ tokens
- Each operation can be optionally accompanied by a broadcast or private message, which will be recorded alongside the transfer on-chain
- FireFly tracks a history of all token operations along with all current token balances
- The blockchain backing each token connector may be the same _or_ different from the one backing FireFly message pinning

## What is a pool?

Token pools are a FireFly construct for describing a set of tokens. The exact definition of a token pool
is dependent on the token connector implementation. Some examples of how pools might map to various well-defined
Ethereum standards:

- **[ERC-1155](https://eips.ethereum.org/EIPS/eip-1155):** a single contract instance can efficiently allocate
  many isolated pools of fungible or non-fungible tokens
  (see [reference implementation](https://github.com/hyperledger/firefly-tokens-erc1155))
- **[ERC-20](https://eips.ethereum.org/EIPS/eip-20) / [ERC-777](https://eips.ethereum.org/EIPS/eip-777):**
  each contract instance represents a single fungible pool of value, e.g. "a coin"
- **[ERC-721](https://eips.ethereum.org/EIPS/eip-721):** each contract instance represents a single pool of NFTs,
  each with unique identities within the pool
- **[ERC-1400](https://github.com/ethereum/eips/issues/1411) / [ERC-1410](https://github.com/ethereum/eips/issues/1410):**
  partially supported in the same manner as ERC-20/ERC-777, but would require new features for working with partitions

These are provided as examples only - a custom token connector could be backed by any token technology (Ethereum or otherwise)
as long as it can support the basic operations described here (create pool, mint, burn, transfer).

## Create a pool

Every application will need to create at least one token pool. At a minimum, you must always
specify a `name` and `type` (fungible or nonfungible) for the pool.

`POST` `/api/v1/namespaces/default/tokens/pools`

```json
{
  "name": "testpool",
  "type": "fungible"
}
```

Other parameters:
- You must specify a `connector` if you have configured multiple token connectors
- You may pass through a `config` object of additional parameters, if supported by your token connector
- You may specify a `key` understood by the connector (i.e. an Ethereum address) if you'd like to use a non-default signing identity

## Mint tokens

Once you have a token pool, you can mint tokens within it. With the default `firefly-tokens-erc1155` connector,
only the creator of a pool is allowed to mint - but each connector may define its own permission model.

`POST` `/api/v1/namespaces/default/tokens/mint`

```json
{
  "amount": 10
}
```

Other parameters:
- You must specify a `pool` name if you've created more than one pool
- You may specify a `key` understood by the connector (i.e. an Ethereum address) if you'd like to use a non-default signing identity
- You may specify `to` if you'd like to send the minted tokens to a specific identity (default is the same as `key`)

## Transfer tokens

You may transfer tokens within a pool by specifying an amount and a destination understood by the connector (i.e. an Ethereum address).
With the default `firefly-tokens-erc1155` connector, only the owner of a token may transfer it away - but each connector may define its
own permission model.

`POST` `/api/v1/namespaces/default/tokens/transfers`

```json
{
  "amount": 1,
  "to": "0x07eab7731db665caf02bc92c286f51dea81f923f"
}
```

Other parameters:
- You must specify a `pool` name if you've created more than one pool
- You must specify a `tokenIndex` for non-fungible pools (and the `amount` should be 1)
- You may specify a `key` understood by the connector (i.e. an Ethereum address) if you'd like to use a non-default signing identity
- You may specify `from` if you'd like to send tokens from a specific identity (default is the same as `key`)

## Sending data with a transfer

All transfers (as well as mint/burn operations) support an optional `message` parameter that contains a broadcast or private
message to be sent along with the transfer. This message follows the same convention as other FireFly messages, and may be comprised
of text or blob data, and can provide context, metadata, or other supporting information about the transfer. The message will be
batched, hashed, and pinned to the primary blockchain as described in [key concepts](../keyconcepts/broadcast.html).

The message ID and hash will also be sent to the token connector as part of the transfer operation, to be written to the token blockchain
when the transaction is submitted. All recipients of the message will then be able to correlate the message with the token transfer.

`POST` `/api/v1/namespaces/default/tokens/transfers`

Broadcast message:
```json
{
  "amount": 1,
  "to": "0x07eab7731db665caf02bc92c286f51dea81f923f",
  "message": {
    "data": [{
      "value": "payment for goods"
    }]
  }
}
```

Private message:
```json
{
  "amount": 1,
  "to": "0x07eab7731db665caf02bc92c286f51dea81f923f",
  "message": {
    "header": {
      "type": "transfer_private",
    },
    "group": {
      "members": [{
          "identity": "org_1"
      }]
    },
    "data": [{
      "value": "payment for goods"
    }]
  }
}
```

Note that all parties in the network will be able to see the transfer (including the message ID and hash), but only
the recipients of the message will be able to view the actual message data.

## Burn tokens

You may burn tokens by simply specifying an amount. With the default `firefly-tokens-erc1155` connector, only the owner of a token may
burn it - but each connector may define its own permission model.

`POST` `/api/v1/namespaces/default/tokens/burn`

```json
{
  "amount": 1,
}
```

Other parameters:
- You must specify a `pool` name if you've created more than one pool
- You must specify a `tokenIndex` for non-fungible pools (and the `amount` should be 1)
- You may specify a `key` understood by the connector (i.e. an Ethereum address) if you'd like to use a non-default signing identity
- You may specify `from` if you'd like to burn tokens from a specific identity (default is the same as `key`)
