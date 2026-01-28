# Tag Ontology and Normalization System

## Overview

A hierarchical tag ontology system that enables consistent tagging, synonym resolution, and intelligent tag-based retrieval.

## Tag Ontology Structure

```mermaid
flowchart TB
    subgraph TagOntology["TAG ONTOLOGY HIERARCHY"]
        direction TB

        subgraph RootCategories["ROOT CATEGORIES"]
            RC1["TECHNICAL"]
            RC2["PERSONAL"]
            RC3["PROJECT"]
            RC4["WORKFLOW"]
            RC5["REFERENCE"]
        end

        subgraph TechnicalTree["TECHNICAL SUBTREE"]
            T1["authentication"]
            T1_1["jwt"]
            T1_2["oauth"]
            T1_3["session"]
            T1_4["cookie"]

            T2["database"]
            T2_1["sql"]
            T2_2["nosql"]
            T2_3["sqlite"]
            T2_4["postgresql"]

            T3["api"]
            T3_1["rest"]
            T3_2["graphql"]
            T3_3["mcp"]

            T4["architecture"]
            T4_1["microservices"]
            T4_2["monolith"]
            T4_3["serverless"]

            RC1 --> T1 --> T1_1
            T1 --> T1_2
            T1 --> T1_3
            T1 --> T1_4
            RC1 --> T2 --> T2_1
            T2 --> T2_2
            T2 --> T2_3
            T2 --> T2_4
            RC1 --> T3 --> T3_1
            T3 --> T3_2
            T3 --> T3_3
            RC1 --> T4 --> T4_1
            T4 --> T4_2
            T4 --> T4_3
        end

        subgraph PersonalTree["PERSONAL SUBTREE"]
            P1["preferences"]
            P1_1["ui-preference"]
            P1_2["workflow-preference"]
            P1_3["communication-style"]

            P2["facts"]
            P2_1["location"]
            P2_2["role"]
            P2_3["organization"]

            RC2 --> P1 --> P1_1
            P1 --> P1_2
            P1 --> P1_3
            RC2 --> P2 --> P2_1
            P2 --> P2_2
            P2 --> P2_3
        end

        subgraph WorkflowTree["WORKFLOW SUBTREE"]
            W1["sop"]
            W1_1["development-sop"]
            W1_2["testing-sop"]
            W1_3["deployment-sop"]

            W2["process"]
            W2_1["git-workflow"]
            W2_2["code-review"]
            W2_3["ci-cd"]

            RC4 --> W1 --> W1_1
            W1 --> W1_2
            W1 --> W1_3
            RC4 --> W2 --> W2_1
            W2 --> W2_2
            W2 --> W2_3
        end
    end

    classDef root fill:#e1f5fe,stroke:#01579b,color:#01579b
    classDef level1 fill:#f3e5f5,stroke:#7b1fa2,color:#7b1fa2
    classDef level2 fill:#e8f5e9,stroke:#2e7d32,color:#1b5e20
    classDef level3 fill:#fff3e0,stroke:#ef6c00,color:#bf360c

    class RC1,RC2,RC3,RC4,RC5 root
    class T1,T2,T3,T4,P1,P2,W1,W2 level1
    class T1_1,T1_2,T1_3,T1_4,T2_1,T2_2,T2_3,T2_4,T3_1,T3_2,T3_3,T4_1,T4_2,T4_3,P1_1,P1_2,P1_3,P2_1,P2_2,P2_3,W1_1,W1_2,W1_3,W2_1,W2_2,W2_3 level2
```

## Synonym Mapping System

```mermaid
flowchart TB
    subgraph SynonymSystem["SYNONYM MAPPING SYSTEM"]
        direction TB

        subgraph SynonymRegistry["SYNONYM REGISTRY"]
            SR1["Canonical Form → Synonyms Map"]
            SR2["jwt → [json-web-token, jsonwebtoken, jwt-token]"]
            SR3["oauth → [oauth2, oauth-2.0, open-auth]"]
            SR4["api → [endpoint, rest-api, web-api]"]
            SR5["database → [db, datastore, data-store]"]
            SR6["authentication → [auth, authn, login]"]
        end

        subgraph NormalizationPipeline["NORMALIZATION PIPELINE"]
            NP1["Input: raw tag string"]
            NP2["Lowercase conversion"]
            NP3["Trim whitespace"]
            NP4["Replace spaces → hyphens"]
            NP5["Remove special characters"]
            NP6["Lookup in synonym registry"]
            NP7{Synonym found?}
            NP8["Return canonical form"]
            NP9["Return normalized input"]
        end

        subgraph Examples["NORMALIZATION EXAMPLES"]
            EX1["'JSON Web Token' → 'jwt'"]
            EX2["'OAuth 2.0' → 'oauth'"]
            EX3["'Data Base' → 'database'"]
            EX4["'REST API' → 'api'"]
            EX5["'Authentication' → 'authentication'"]
        end
    end

    NP1 --> NP2 --> NP3 --> NP4 --> NP5 --> NP6 --> NP7
    NP7 -->|Yes| NP8
    NP7 -->|No| NP9

    classDef registry fill:#e1f5fe,stroke:#01579b,color:#01579b
    classDef pipeline fill:#f3e5f5,stroke:#7b1fa2,color:#7b1fa2
    classDef example fill:#e8f5e9,stroke:#2e7d32,color:#1b5e20

    class SR1,SR2,SR3,SR4,SR5,SR6 registry
    class NP1,NP2,NP3,NP4,NP5,NP6,NP7,NP8,NP9 pipeline
    class EX1,EX2,EX3,EX4,EX5 example
```

## Tag Enrichment Pipeline

```mermaid
flowchart TB
    subgraph TagEnrichment["TAG ENRICHMENT PIPELINE"]
        direction TB

        subgraph Input["INPUT"]
            I1["Normalized tag"]
            I2["Memory content"]
            I3["Tag ontology"]
        end

        subgraph HierarchyLookup["HIERARCHY LOOKUP"]
            HL1["Find tag in ontology tree"]
            HL2{Tag exists in ontology?}
            HL3["Get parent chain"]
            HL4["Get sibling tags (optional)"]
            HL5["Add as new leaf node"]
            HL6["Suggest parent category"]
        end

        subgraph ParentEnrichment["PARENT TAG ENRICHMENT"]
            PE1["For each parent in chain:"]
            PE2["Add parent as implicit tag"]
            PE3["Limit depth (max 2 levels up)"]
            PE4["Example: 'jwt' adds 'authentication', 'technical'"]
        end

        subgraph RelatedTags["RELATED TAG SUGGESTIONS"]
            RT1["Query memories with same tag"]
            RT2["Find frequently co-occurring tags"]
            RT3["Co-occurrence > threshold (0.3)"]
            RT4["Suggest related tags"]
        end

        subgraph WeightedTags["TAG WEIGHTING"]
            WT1["Assign weights based on source:"]
            WT2["Explicit user tag: weight = 1.0"]
            WT3["Auto-extracted tag: weight = 0.8"]
            WT4["Parent enrichment: weight = 0.5"]
            WT5["Co-occurrence suggestion: weight = 0.3"]
        end

        subgraph Output["OUTPUT"]
            O1["Enriched tag set with weights"]
            O2["Primary tags (weight >= 0.8)"]
            O3["Secondary tags (weight < 0.8)"]
        end
    end

    I1 --> HL1
    I3 --> HL1
    HL1 --> HL2
    HL2 -->|Yes| HL3 --> HL4
    HL2 -->|No| HL5 --> HL6

    HL3 --> PE1 --> PE2 --> PE3 --> PE4

    I2 --> RT1 --> RT2 --> RT3 --> RT4

    PE4 --> WT1
    RT4 --> WT1
    WT1 --> WT2 --> WT3 --> WT4 --> WT5

    WT5 --> O1 --> O2
    O1 --> O3

    classDef input fill:#e1f5fe,stroke:#01579b,color:#01579b
    classDef hierarchy fill:#f3e5f5,stroke:#7b1fa2,color:#7b1fa2
    classDef parent fill:#e8f5e9,stroke:#2e7d32,color:#1b5e20
    classDef related fill:#fff3e0,stroke:#ef6c00,color:#bf360c
    classDef weight fill:#fce4ec,stroke:#c2185b,color:#880e4f
    classDef output fill:#e0f7fa,stroke:#00838f,color:#00695c

    class I1,I2,I3 input
    class HL1,HL2,HL3,HL4,HL5,HL6 hierarchy
    class PE1,PE2,PE3,PE4 parent
    class RT1,RT2,RT3,RT4 related
    class WT1,WT2,WT3,WT4,WT5 weight
    class O1,O2,O3 output
```

## Tag-Based Retrieval Enhancement

```mermaid
flowchart TB
    subgraph TagRetrieval["TAG-BASED RETRIEVAL ENHANCEMENT"]
        direction TB

        subgraph QueryAnalysis["QUERY ANALYSIS"]
            QA1[/"User query"/]
            QA2["Extract explicit tags from query"]
            QA3["Infer implicit tags from content"]
            QA4["Expand tags via ontology"]
            QA5["Build tag filter set"]
        end

        subgraph TagExpansion["TAG EXPANSION"]
            TE1["For each query tag:"]
            TE2["Add synonyms"]
            TE3["Add child tags (narrow)"]
            TE4["Optionally add parent tags (broaden)"]
            TE5["Expanded tag set"]
        end

        subgraph FilteredRetrieval["FILTERED RETRIEVAL"]
            FR1["Vector similarity search"]
            FR2["Apply tag filter (AND/OR logic)"]
            FR3{Filter mode?}
            FR4["STRICT: must have ALL tags"]
            FR5["LOOSE: must have ANY tag"]
            FR6["WEIGHTED: boost by tag match count"]
        end

        subgraph ScoreAdjustment["SCORE ADJUSTMENT"]
            SA1["base_score = vector_similarity"]
            SA2["tag_match_count = count overlapping tags"]
            SA3["tag_boost = tag_match_count * 0.1"]
            SA4["final_score = base_score + tag_boost"]
            SA5["Cap at 1.0"]
        end

        subgraph Output["OUTPUT"]
            O1["Sort by final_score DESC"]
            O2["Tag-enriched results"]
            O3["Highlight matching tags"]
        end
    end

    QA1 --> QA2 --> QA3 --> QA4 --> QA5

    QA5 --> TE1 --> TE2 --> TE3 --> TE4 --> TE5

    TE5 --> FR1 --> FR2 --> FR3
    FR3 -->|Strict| FR4
    FR3 -->|Loose| FR5
    FR3 -->|Weighted| FR6

    FR4 --> SA1
    FR5 --> SA1
    FR6 --> SA1

    SA1 --> SA2 --> SA3 --> SA4 --> SA5

    SA5 --> O1 --> O2 --> O3

    classDef query fill:#e1f5fe,stroke:#01579b,color:#01579b
    classDef expand fill:#f3e5f5,stroke:#7b1fa2,color:#7b1fa2
    classDef filter fill:#e8f5e9,stroke:#2e7d32,color:#1b5e20
    classDef score fill:#fff3e0,stroke:#ef6c00,color:#bf360c
    classDef output fill:#fce4ec,stroke:#c2185b,color:#880e4f

    class QA1,QA2,QA3,QA4,QA5 query
    class TE1,TE2,TE3,TE4,TE5 expand
    class FR1,FR2,FR3,FR4,FR5,FR6 filter
    class SA1,SA2,SA3,SA4,SA5 score
    class O1,O2,O3 output
```

## Ontology Evolution and Learning

```mermaid
flowchart TB
    subgraph OntologyEvolution["ONTOLOGY EVOLUTION"]
        direction TB

        subgraph NewTagDetection["NEW TAG DETECTION"]
            NT1["Monitor incoming tags"]
            NT2{Tag exists in ontology?}
            NT3["Use existing placement"]
            NT4["Queue for classification"]
        end

        subgraph AutoClassification["AUTO-CLASSIFICATION"]
            AC1["Collect unclassified tags"]
            AC2["LLM prompt: Suggest parent category"]
            AC3["'Given tag: {tag}<br/>Existing categories: {ontology}<br/>Which category fits best?'"]
            AC4["Parse suggested parent"]
            AC5{Confidence > 0.8?}
            AC6["Auto-add to ontology"]
            AC7["Queue for human review"]
        end

        subgraph UsageAnalysis["USAGE ANALYSIS"]
            UA1["Periodic analysis job"]
            UA2["Count tag usage frequencies"]
            UA3["Identify underused tags"]
            UA4["Identify missing parent categories"]
            UA5["Suggest ontology improvements"]
        end

        subgraph SynonymDiscovery["SYNONYM DISCOVERY"]
            SD1["Analyze tags that always co-occur"]
            SD2["co-occurrence > 0.9 = potential synonym"]
            SD3["LLM verification: Are these synonyms?"]
            SD4{Confirmed synonym?}
            SD5["Add to synonym registry"]
            SD6["Keep as distinct tags"]
        end

        subgraph OntologyMaintenance["MAINTENANCE"]
            OM1["Prune unused tags (0 memories)"]
            OM2["Merge near-identical branches"]
            OM3["Rebalance deep hierarchies"]
            OM4["Export ontology as JSON/YAML"]
        end
    end

    NT1 --> NT2
    NT2 -->|Yes| NT3
    NT2 -->|No| NT4

    NT4 --> AC1 --> AC2 --> AC3 --> AC4 --> AC5
    AC5 -->|Yes| AC6
    AC5 -->|No| AC7

    AC6 --> UA1
    UA1 --> UA2 --> UA3 --> UA4 --> UA5

    UA5 --> SD1 --> SD2 --> SD3 --> SD4
    SD4 -->|Yes| SD5
    SD4 -->|No| SD6

    SD5 --> OM1 --> OM2 --> OM3 --> OM4

    classDef detect fill:#e1f5fe,stroke:#01579b,color:#01579b
    classDef classify fill:#f3e5f5,stroke:#7b1fa2,color:#7b1fa2
    classDef usage fill:#e8f5e9,stroke:#2e7d32,color:#1b5e20
    classDef synonym fill:#fff3e0,stroke:#ef6c00,color:#bf360c
    classDef maintain fill:#fce4ec,stroke:#c2185b,color:#880e4f

    class NT1,NT2,NT3,NT4 detect
    class AC1,AC2,AC3,AC4,AC5,AC6,AC7 classify
    class UA1,UA2,UA3,UA4,UA5 usage
    class SD1,SD2,SD3,SD4,SD5,SD6 synonym
    class OM1,OM2,OM3,OM4 maintain
```

## Tag Data Model

```mermaid
classDiagram
    class Tag {
        +string id
        +string canonical_name
        +string[] synonyms
        +string parent_id
        +int depth
        +int usage_count
        +datetime created_at
        +datetime last_used
    }

    class TagOntology {
        +Tag[] tags
        +map~string,string~ synonym_to_canonical
        +addTag(name, parent_id)
        +normalize(input) string
        +getAncestors(tag_id) Tag[]
        +getDescendants(tag_id) Tag[]
        +getSynonyms(tag_id) string[]
    }

    class MemoryTag {
        +string memory_id
        +string tag_id
        +float weight
        +string source
        +datetime assigned_at
    }

    class TagCooccurrence {
        +string tag_a_id
        +string tag_b_id
        +int cooccur_count
        +float cooccur_ratio
    }

    Tag "1" --> "*" Tag : parent_of
    TagOntology "1" --> "*" Tag : contains
    MemoryTag "*" --> "1" Tag : references
    TagCooccurrence "*" --> "2" Tag : relates
```

## Tag Storage Schema (SQLite)

```mermaid
erDiagram
    tags {
        TEXT id PK
        TEXT canonical_name UK
        TEXT parent_id FK
        INTEGER depth
        INTEGER usage_count
        TEXT synonyms_json
        DATETIME created_at
        DATETIME last_used
    }

    memory_tags {
        TEXT memory_id FK
        TEXT tag_id FK
        REAL weight
        TEXT source
        DATETIME assigned_at
    }

    tag_cooccurrence {
        TEXT tag_a_id FK
        TEXT tag_b_id FK
        INTEGER cooccur_count
        REAL cooccur_ratio
    }

    memories {
        TEXT id PK
        TEXT content
        TEXT tags_json
    }

    tags ||--o{ tags : "parent_of"
    memories ||--o{ memory_tags : "has"
    tags ||--o{ memory_tags : "applied_to"
    tags ||--o{ tag_cooccurrence : "cooccurs_with"
```

## Tag Query Examples

```mermaid
flowchart LR
    subgraph QueryExamples["TAG QUERY EXAMPLES"]
        direction TB

        subgraph Example1["QUERY 1: Strict Tag Filter"]
            Q1[/"Find memories tagged 'jwt' AND 'authentication'"/]
            SQL1["SELECT * FROM memories m<br/>JOIN memory_tags mt1 ON m.id = mt1.memory_id<br/>JOIN memory_tags mt2 ON m.id = mt2.memory_id<br/>WHERE mt1.tag_id = 'jwt'<br/>AND mt2.tag_id = 'authentication'"]
        end

        subgraph Example2["QUERY 2: Hierarchical Search"]
            Q2[/"Find memories in 'technical' category (any depth)"/]
            SQL2["WITH RECURSIVE subtree AS (<br/>  SELECT id FROM tags WHERE id = 'technical'<br/>  UNION ALL<br/>  SELECT t.id FROM tags t<br/>  JOIN subtree s ON t.parent_id = s.id<br/>)<br/>SELECT * FROM memories m<br/>JOIN memory_tags mt ON m.id = mt.memory_id<br/>WHERE mt.tag_id IN (SELECT id FROM subtree)"]
        end

        subgraph Example3["QUERY 3: Synonym-Aware Search"]
            Q3[/"Find memories tagged 'json-web-token' (synonym of 'jwt')"/]
            STEP1["1. Normalize 'json-web-token' → 'jwt'"]
            STEP2["2. Query for canonical tag 'jwt'"]
            SQL3["SELECT * FROM memories m<br/>JOIN memory_tags mt ON m.id = mt.memory_id<br/>WHERE mt.tag_id = 'jwt'"]
        end

        subgraph Example4["QUERY 4: Tag Suggestions"]
            Q4[/"Suggest tags for new memory about OAuth2"/]
            STEP4_1["1. Extract keywords: 'oauth', 'oauth2'"]
            STEP4_2["2. Normalize: 'oauth'"]
            STEP4_3["3. Lookup hierarchy: 'oauth' → 'authentication' → 'technical'"]
            STEP4_4["4. Find co-occurring tags: 'security', 'api'"]
            RESULT4["Suggested: oauth, authentication, technical, security, api"]
        end
    end

    Q1 --> SQL1
    Q2 --> SQL2
    Q3 --> STEP1 --> STEP2 --> SQL3
    Q4 --> STEP4_1 --> STEP4_2 --> STEP4_3 --> STEP4_4 --> RESULT4

    classDef query fill:#e1f5fe,stroke:#01579b,color:#01579b
    classDef sql fill:#f3e5f5,stroke:#7b1fa2,color:#7b1fa2
    classDef step fill:#e8f5e9,stroke:#2e7d32,color:#1b5e20
    classDef result fill:#fff3e0,stroke:#ef6c00,color:#bf360c

    class Q1,Q2,Q3,Q4 query
    class SQL1,SQL2,SQL3 sql
    class STEP1,STEP2,STEP4_1,STEP4_2,STEP4_3,STEP4_4 step
    class RESULT4 result
```
