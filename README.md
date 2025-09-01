# Dice-Sorensen Similarity Search

this Go application comprises the following features
- fetch data from a Bitbucket repository
- normalize fetched Markdowns and save them into a PostgreSQL database
- build a tree structure that corresponds to the folder structure in the Bitbucket repository
- load the content of a Markdown file by it's file name
- search through Markdowns that were already saved into the database (based on Dice-Sorensen similarity) 