# Wallet Service Implementation Guide

## Overview

A robust wallet service implementing double-entry bookkeeping with atomic transactions, optimistic locking, and gamified virtual deposits.

## Table of Contents

- [Core Features](#core-features)
- [Database Schema](#database-schema)
- [API Endpoints](#api-endpoints)
- [Key Flows](#key-flows)
  - [Wallet-to-Wallet Transfer](#1-wallet-to-wallet-transfer)
  - [Gamified Deposit](#2-gamified-deposit)
- [Concurrency Control](#concurrency-control)
- [Event-Driven Architecture](#event-driven-architecture)
- [Local Setup](#local-setup)
- [Demo Scenarios](#demo-scenarios)

## Core Features

✅ **Double-Entry Bookkeeping**

- Every transaction records both debit (source) and credit (destination)
- Ensures real-time balance consistency

✅ **Optimistic Locking**

- Versioned updates prevent race conditions
- Uses version numbers for concurrent balance modifications

✅ **Virtual Money Deposits**

- Gamified onboarding (quizzes, mini-games)
- Fun ways to earn demo currency

✅ **Transaction Processing**

- Atomic debit/credit operations
- Idempotency keys for duplicate prevention

✅ **Compliance Mocking**

- Auto-verification after delay (demo only)
