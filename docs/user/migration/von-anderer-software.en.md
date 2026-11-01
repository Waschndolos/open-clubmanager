# Switching from other club software

There is no direct import from other club programs yet. The way in is an **Excel file**: almost every club program can export its member list as Excel or CSV.

## General procedure

1. **Export from the old software:** export the member list with all fields as Excel (`.xlsx`) or CSV. Include bank details and SEPA mandates if you use direct debits.
2. **Open the CSV in Excel/LibreOffice and save it as `.xlsx`.** Watch the character encoding (UTF-8) when opening, so accented characters come out right.
3. **Prepare it** as described under [Switching from Excel](von-excel.md#1-preparing-the-excel-list).
4. **Import** and **check**.
5. **Run in parallel:** keep the old software running read-only for a while (e.g. until the next fee collection) before cancelling it.

## What cannot be transferred

- **Transaction history** of past years: close the old financial year in the old software and start in Open ClubManager with the opening balance.
- **Documents and attachments** (scans, membership applications): Open ClubManager doesn't store files. Keep them in the club archive.
- **Online access for members**: doesn't exist in Open ClubManager.

## Guides for specific programs

!!! info "Ready for community contributions"
    This is where we collect step-by-step guides for individual programs. You switched from another software? Add a guide; the template is below and in [Maintaining the documentation](../mitwirken/dokumentation.md).

| Software | Status |
|---|---|
| *none yet* | – |

??? abstract "Template for a guide"
    ```markdown
    ### From <name of the software>

    **Tested with:** <version of the old software>, Open ClubManager <version>

    1. Export: <menu path in the old software>, format <xlsx/csv>
    2. File specifics: <separator, encoding, date format>
    3. Column mapping:

    | Column in the export | Field in Open ClubManager |
    |---|---|
    | … | … |

    4. What's missing / follow-up: <…>
    ```
